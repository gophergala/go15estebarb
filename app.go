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
)

var processImage = delay.Func("key", doProcessing)

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
	processImage.Call(c, file[0].BlobKey)
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

func doProcessing(c appengine.Context, blobkey appengine.BlobKey){
	r := blobstore.NewReader(c, blobkey)
	img, _, err := image.Decode(r)
	if err != nil{
		c.Errorf("%v", err)
	}
	img = filterGrayscale(img)
	saveImage(c, img)
}

func saveImage(c appengine.Context, m image.Image){
	w, err := blobstore.Create(c, "image/png")
	if err != nil{
		c.Errorf("%v", err)
	}
	defer w.Close()
	
	png.Encode(w, m)
}

func filterGrayscale(m image.Image) image.Image{
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

func distance(A, B []int) int{
	dy := A[1]-B[1]
	dx := A[0]-B[0]
	return int(math.Sqrt(float64(dy*dy+dx*dx)))
}

func manhattan(A, B []int) int{
	dy := A[1]-B[1]
	dx := A[0]-B[0]
	return int(math.Abs(float64(dy))+math.Abs(float64(dx)))
}

func colorMean(colors []color.Color)color.Color{
	r,g,b := 0,0,0
	c := 0
	for v:= range colors{
		R, G, B, _ := v.RGBA()
		r += R
		g += G
		b += B
		c++
	}
	return color.NRGBA{r/c, r/c, r/c, 0}
}

func filterVoronoi(m image.Image) image.Image{
	out := imaging.Clone(m)
	bounds := out.Bounds()
	numClusters := int(math.Sqrt(bounds.Max.Y * bounds.Max.X))
	// Generates the centroids
	centroids := make(map[int]([]int))
	for i := 0; i < numClusters; i++{
		centroids[i] = 	[]int{rand.Intn(bounds.Max.X), rand.Intn(bounds.Max.Y)}
	}
	maxval := numClusters*numClusters*numClusters
	clSelection := bidimensionalArray(bounds.Max.X, bounds.Max.Y)
	clusterColors := make(map[int]([]color.Color))
	
	// Finds the nearest cluster	
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			mindist := maxval
			minCentroid := 0
			for i := 0; i < numClusters; i++ {
				clDistance := distance(centroids[i], []int{x, y})
				if clDistance < mindist {
					mindist = clDistance
					minCentroid=i
				}
			}
			// add the colors to the cluster colors selection
			clSelection[x][y] = minCentroid
			append(clusterColors[minCentroid], m.At(x,y))
		}
	}
	
	// Averages colors
	finalColors := make([]color.Color, numClusters)
	for k, v := range clusterColors{
		if len(v)>0{
			finalColors[k] = colorMean(v)
		}
	}
	
	// Writes image
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := m.At(x, y).RGBA()

		}
	}
}


