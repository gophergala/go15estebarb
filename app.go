package gophergala

import (
	"net/http"
	"html/template"
	"io"
	"math"
	"math/rand"
	"image"
	"image/color"
	 "image/png"
	_ "image/jpeg"
	"appengine"
	"appengine/blobstore"
	"appengine/delay"
	"github.com/disintegration/imaging"
	"code.google.com/p/draw2d/draw2d"
)

var processImageGrayscale = delay.Func("grayscale", doProcessingGray)
var processImageVoronoi = delay.Func("voronoi", doProcessingVoronoi)
var processImageOilPaint = delay.Func("oilpaint", doProcessingOilPaint)
var processImagePainterly = delay.Func("painterly", doProcessingPainterly)


func init() {
	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/serve/", handleServe)
	http.HandleFunc("/", handler)
}

func serveError(c appengine.Context, w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "Internal Server Error")
	c.Errorf("%v", err)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	blobs, _, err := blobstore.ParseUpload(r)
	if err != nil {
		serveError(c, w, err)
		return
	}
	
	file := blobs["file"]
	if len(file) == 0 {
		c.Errorf("no file uploaded")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	processImageGrayscale.Call(c, file[0].BlobKey)
	processImageVoronoi.Call(c, file[0].BlobKey)
	processImageOilPaint.Call(c, file[0].BlobKey)
	processImagePainterly.Call(c, file[0].BlobKey)
	http.Redirect(w, r, "/serve/?blobKey="+string(file[0].BlobKey), http.StatusFound)
}

func handleServe(w http.ResponseWriter, r *http.Request) {
	blobstore.Send(w, appengine.BlobKey(r.FormValue("blobKey")))
}

func handler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	uploadURL, err := blobstore.UploadURL(c, "/upload", nil)
	t, _ := template.ParseFiles("templates/home.html")
	t.Execute(w, uploadURL)
	if err != nil {
		c.Errorf("%v", err)
	}
}

func rescaleImage(m image.Image)image.Image{
	bounds := m.Bounds()
	ys := bounds.Max.Y
	xs := bounds.Max.X
	if xs > ys{
		return imaging.Resize(m, IntMin(800, xs), 0, imaging.Lanczos)
	}else{
		return imaging.Resize(m, 0, IntMin(800, ys), imaging.Lanczos)
	}
}

func doProcessingGray(c appengine.Context, blobkey appengine.BlobKey){
	r := blobstore.NewReader(c, blobkey)
	img, _, err := image.Decode(r)
	if err != nil{
		c.Errorf("%v", err)
	}
	img = rescaleImage(img)
	pic := filterGrayscale(c, img)
	saveImage(c, pic)
}

func doProcessingVoronoi(c appengine.Context, blobkey appengine.BlobKey){
	r := blobstore.NewReader(c, blobkey)
	img, _, err := image.Decode(r)
	if err != nil {
		c.Errorf("%v", err)
	}
	img = rescaleImage(img)
	pic := filterVoronoi(c, img)
	saveImage(c, pic)
}

func doProcessingOilPaint(c appengine.Context, blobkey appengine.BlobKey){
	r := blobstore.NewReader(c, blobkey)
	img, _, err := image.Decode(r)
	if err != nil {
		c.Errorf("%v", err)
	}
	img = rescaleImage(img)
	pic := filterOilPaint(c, img)
	saveImage(c, pic)
}


func doProcessingPainterly(c appengine.Context, blobkey appengine.BlobKey){
	r := blobstore.NewReader(c, blobkey)
	img, _, err := image.Decode(r)
	if err != nil {
		c.Errorf("%v", err)
	}
	img = rescaleImage(img)
	pic := filterPainterly(c, img)
	saveImage(c, pic)
}

func saveImage(c appengine.Context, m image.Image){
	w, err := blobstore.Create(c, "image/png")
	if err != nil{
		c.Errorf("%v", err)
	}
	defer w.Close()
	png.Encode(w, m)
}

func filterGrayscale(_ appengine.Context, m image.Image) image.Image{
	res := imaging.Grayscale(m)
	return res
}

func bidimensionalArray(x, y int) [][]int{
	res := make([][]int, x)
	for i := range res{
		res[i] = make([]int, y)
	}
	return res
}

func distance(A []int, x, y int) float64{
	dy := A[1]-y
	dx := A[0]-x
	return math.Sqrt(float64(dy*dy+dx*dx))
}

func manhattan(A []int, x, y int) float64{
	dy := A[1]-y
	dx := A[0]-x
	return math.Abs(float64(dy))+math.Abs(float64(dx))
}

func colorMean(colors []color.Color)color.Color{
	var r,g,b,a float64
	r,g,b,a = 0,0,0,0
	for _, v := range colors{
		R, G, B, A := v.RGBA()
		r += float64(R)
		g += float64(G)
		b += float64(B)
		a += float64(A)
	}
	c := float64(len(colors))
	return color.NRGBA{uint8(r/c), uint8(g/c), uint8(b/c), uint8(a/c)}
}

type MyColor struct{
	R, G, B, A, C int64
}

func (o *MyColor) Add(c color.Color){
	r, g, b, a := c.RGBA()
	o.R+=int64(r)
	o.G+=int64(g)
	o.B+=int64(b)
	o.A+=int64(a)
	o.C++
}

func (o *MyColor) Average() color.Color{
	if o.C == 0{
		return color.Black
	}
	return color.NRGBA64{R:uint16(o.R/o.C), G:uint16(o.G/o.C), B:uint16(o.B/o.C), A:uint16(o.A/o.C)}
}

func filterVoronoi(c appengine.Context, m image.Image) image.Image{
	bounds := m.Bounds()
	out := image.NewNRGBA(bounds)
	numClusters := int(math.Sqrt(float64(bounds.Max.Y * bounds.Max.X)))
	// Generates the centroids
	centroids := make(map[int]([]int))
	for i := 0; i < numClusters; i++{
		centroids[i] = 	[]int{rand.Intn(bounds.Max.X), rand.Intn(bounds.Max.Y)}
	}
	maxval := float64(numClusters*numClusters*numClusters)
	//clSelection := bidimensionalArray(bounds.Max.X, bounds.Max.Y)
	clusterColors := make(map[int]MyColor)
	
	// Finds the nearest cluster
	clSelection := make([][]int, bounds.Max.Y)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		rowSelection := make([]int, bounds.Max.X)
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			mindist := maxval
			minCentroid := 0
			for i := 0; i < numClusters; i++ {
				clDistance := distance(centroids[i], x, y)
				if clDistance < mindist {
					mindist = clDistance
					minCentroid=i
				}
			}
			// add the colors to the cluster colors selection
			//r, g, b, a := m.At(x,y).RGBA()
			//c.Infof("Color %v %v %v %v (%v)", r, g, b, a, m.At(x,y))
			
			rowSelection[x] = minCentroid
			curColor := clusterColors[minCentroid]
			curColor.Add(m.At(x,y))
			clusterColors[minCentroid] = curColor
		}
		clSelection[y] = rowSelection
	}
	
	// Averages colors
	finalColors := make([]color.Color, numClusters)
	for k, v := range clusterColors{
			//finalColors[k] = m.At(centroids[k][0], centroids[k][1])//colorMean(v)
			finalColors[k] = v.Average()
	}
	
	// Writes image
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			out.Set(x,y, finalColors[clSelection[y][x]])
		}
	}
	return out
}

func IntMax(a, b int)int{
	if a > b{
		return a		
	}	
	return b
}

func IntMin(a, b int)int{
	if a > b{
		return b
	}
	return a
}

func filterOilPaint(c appengine.Context, m image.Image) image.Image{
	bounds := m.Bounds()
	out := image.NewNRGBA(bounds)
	ys := bounds.Max.Y
	xs := bounds.Max.X
	radius := 5
	intensityLevels := 20
	
	intensityMap := make([][]uint8, ys)
	for y := 0; y < ys; y++ {
		intensityRow := make([]uint8, xs)
		for x := 0; x < xs; x++ {
			currentColor := m.At(x, y)
			r,g,b,_ := currentColor.RGBA()
			//c.Infof("Color %v %v %v (%v)", r, g, b, currentColor)
			ci := uint8(int(r+g+b)/3.0*intensityLevels/255.0/255.0)
			intensityRow[x] = ci
		}
		intensityMap[y] = intensityRow
	}
	
	for y := 0; y < ys; y++ {
		for x := 0; x < xs; x++ {
			intensities := make([]MyColor, intensityLevels+1)
			for y2 := IntMax(0, y-radius); y2 < IntMin(ys, y+radius); y2++ {
				for x2 := IntMax(0, x-radius); x2 < IntMin(xs, x+radius); x2++ {
					currentColor := m.At(x2, y2)
					//r,g,b,_ := currentColor.RGBA()
					//c.Infof("Color %v %v %v (%v)", r, g, b, currentColor)
					//ci := int(int(r+g+b)/3.0*intensityLevels/255.0/255.0)
					ci := intensityMap[y2][x2]
					//c.Infof("Intensities %v of %v", ci, len(intensities))
					newColor := intensities[ci]
					newColor.Add(currentColor)
					intensities[ci] = newColor
				}
			}
			newColor := intensities[0]
			for _, v := range intensities{
				if newColor.C < v.C{
					newColor = v
				}
			}
			out.Set(x, y, newColor.Average())
		}
	}
	return out
}

////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////

func generateBrushes(minRad, numBrushes int) []int{
	brushes := make([]int, numBrushes)
	for k := range brushes{
		brushes[numBrushes-k-1] = minRad
		minRad += minRad
	}
	return brushes
}

func filterPainterly(c appengine.Context, m image.Image) image.Image{
	bounds := m.Bounds()
	canvas := image.NewRGBA(bounds)
	
	// Estos parámetros posteriormente deberán ser... parametrizados:
	brushMinRadius := 3
	numOfBrushes := 3
	brushes := generateBrushes(brushMinRadius, numOfBrushes)
	
	for _, radius := range brushes{
		c.Infof("Brush %v", radius)
		refImage := imaging.Blur(m, float64(radius))
		paintLayer(canvas, refImage, radius, 100)
	}
	return canvas
}

func imageDifference(A *image.RGBA, B image.Image)[][]float64{
	ys := A.Bounds().Max.Y
	xs := A.Bounds().Max.X
	res := make([][]float64, ys)
	for y := 0; y < ys; y++{
		rowdif := make([]float64, xs)
		for x:=0; x < xs; x++{
			ar, ag, ab, aa := A.At(x,y).RGBA()
			br, bg, bb, ba := B.At(x,y).RGBA()
			dr, dg, db, da := float64(ar-br), float64(ag-bg), float64(ab-bb), float64(aa-ba)
			rowdif[x] = math.Sqrt(dr*dr+dg*dg+db*db+da*da)
		}
		res[y] = rowdif
	}
	return res
}

type MyStroke struct{
	Color color.Color
	Point image.Point
	Radius int
}

func paintLayer(cnv *image.RGBA, refImage image.Image, radius int, T float64) image.Image{
	strokes := make([]MyStroke,0)
	D := imageDifference(cnv, refImage)

	ys := cnv.Bounds().Max.Y
	xs := cnv.Bounds().Max.X
	for y := 0; y < ys; y++{
		for x:=0; x < xs; x++{
			// Calculates the error near (x,y):
			areaError := float64(0)
			maxdif := float64(0)
			maxx := 0
			maxy := 0
			for y2 := IntMax(0, y-radius); y2 < IntMin(ys, y+radius); y2++ {
				for x2 := IntMax(0, x-radius); x2 < IntMin(xs, x+radius); x2++ {
					dif := D[y2][x2]
					areaError += dif
					if dif > maxdif{
						maxdif = dif
						maxx = x2
						maxy = y2
					}
				}
			}
			areaError = areaError / float64(radius*radius)
			
			if areaError > T {
				strokes = append(strokes, MyStroke{
						Color: refImage.At(maxx, maxy),
						Point: image.Point{maxx, maxy},
						Radius: radius,
					})
			}
		}
	}
	PaintStrokes(cnv, strokes)
	return cnv
}

func PaintStrokes(cnv *image.RGBA, strokes []MyStroke){
	gc := draw2d.NewGraphicContext(cnv)
	order := rand.Perm(len(strokes))
	
	for _, v := range order{
		s := strokes[v]
		gc.SetFillColor(s.Color)
		gc.SetLineWidth(0)
		gc.ArcTo(float64(s.Point.X), 
			float64(s.Point.Y),
				float64(s.Radius),
					float64(s.Radius),
						0, 2*math.Pi)
		gc.FillStroke()
	}
}

