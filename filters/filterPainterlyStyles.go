package filters

import (
	"appengine"
	"code.google.com/p/draw2d/draw2d"
	"github.com/disintegration/imaging"
	"image"
	_ "image/jpeg"
	"math"
)

func generateBrushesStyles(minRad, numBrushes int) []int {
	brushes := make([]int, numBrushes)
	for k := range brushes {
		brushes[numBrushes-k-1] = minRad
		minRad += minRad
	}
	return brushes
}

func FilterPainterlyStyles(c appengine.Context, m image.Image, settings *PainterlySettings) image.Image {
	bounds := m.Bounds()
	canvas := image.NewRGBA(bounds)

	// Estos parámetros posteriormente deberán ser... parametrizados:
	brushes := generateBrushes(settings.Style.Radius, settings.Style.NumOfBrushes)

	for _, radius := range brushes {
		c.Infof("Brush %v", radius)
		refImage := imaging.Blur(m, settings.Style.BlurFactor*float64(radius))
		paintLayerStyles(canvas, refImage, radius, settings)
	}
	return canvas
}

type PainterlySettings struct{
	Style PainterlyStyle
	Blobkey appengine.BlobKey
}

type PainterlyStyle struct{
	// Aproximation threshold
	T float64
	
	// Brushes
	Radius int
	NumOfBrushes int
	
	// Curvature Filter f_c
	FC float64
	
	// Blur Factor ¿f_s?
	BlurFactor float64
	
	// Length of strokes
	MinimumStroke int
	MaximumStroke int
	
	// Opacity alpha
	Opacity float64
	
	// GridSize f_g
	GridSize float64
	
	// Jitter
	JitterHue float64 //j_h
	JitterSaturation float64 //j_s
	JitterValue float64 //j_v
	JitterRed float64
	JitterGreen float64
	JitterBlue float64
}

// A normal painting style, with no curvature filter, and
// no random color. T = 100, R=(8,4,2),
// fc=1, fs=.5, a=1, fg=1, minlen=4 and maxlen=16
var StyleImpressionist = PainterlyStyle{
	T: 100,
	Radius: 2,
	NumOfBrushes: 3,
	FC: 1,
	BlurFactor: 0.5,
	Opacity: 1,
	GridSize: 1,
	MinimumStroke: 4,
	MaximumStroke: 16,
}

var StyleExpressionist = PainterlyStyle{
	T: 50,
	Radius: 2,
	NumOfBrushes: 3,
	FC: 0.25,
	BlurFactor: 0.5,
	Opacity: 0.7,
	GridSize: 1,
	MinimumStroke: 10,
	MaximumStroke: 16,
	JitterValue: 0.5,
}

var StyleColoristWash = PainterlyStyle{
	T: 200,
	Radius: 2,
	NumOfBrushes: 3,
	FC: 1,
	BlurFactor: 0.5,
	Opacity: 0.5,
	GridSize: 1,
	MinimumStroke: 4,
	MaximumStroke: 16,
	JitterRed: 0.3,
	JitterGreen: 0.3,
	JitterBlue: 0.3,
}

var StylePointillist = PainterlyStyle{
	T: 100,
	Radius: 2,
	NumOfBrushes: 2,
	FC: 1,
	BlurFactor: 0.5,
	Opacity: 1,
	GridSize: 0.5,
	MinimumStroke: 0,
	MaximumStroke: 0,
	JitterValue: 1,
	JitterHue: 0.3,
}

var StylePsychedelic = PainterlyStyle{
	T: 50,
	Radius: 2,
	NumOfBrushes: 3,
	FC: 0.5,
	BlurFactor: 0.5,
	Opacity: 0.7,
	GridSize: 1,
	MinimumStroke: 10,
	MaximumStroke: 16,
	JitterHue: 0.5,
	JitterSaturation: 0.25,
}

func paintLayerStyles(cnv *image.RGBA, refImage image.Image, radius int, settings *PainterlySettings) image.Image {
	D := ImageDifference(cnv, refImage)
	gradMag, gradOri := GradientData(refImage)
	ys := cnv.Bounds().Max.Y
	xs := cnv.Bounds().Max.X
	for y := 0; y < ys; y++ {
		for x := 0; x < xs; x++ {
			// Calculates the error near (x,y):
			areaError := float64(0)
			maxdif := float64(0)
			maxx := 0
			maxy := 0
			for y2 := IntMax(0, y-radius); y2 < IntMin(ys, y+radius); y2++ {
				for x2 := IntMax(0, x-radius); x2 < IntMin(xs, x+radius); x2++ {
					dif := D[y2][x2]
					areaError += dif
					if dif > maxdif {
						maxdif = dif
						maxx = x2
						maxy = y2
					}
				}
			}
			areaError = areaError / float64(radius*radius)

			if areaError > settings.Style.T {
				newstroke := createCurve(cnv, refImage, gradMag, gradOri, maxx, maxy, radius,
				settings)
				drawStroke(cnv, newstroke)
			}
		}
	}
	return cnv
}

func drawStroke(cnv *image.RGBA, points []MyStroke){
	if len(points) == 0{
		return
	}
	gc := draw2d.NewGraphicContext(cnv)
	strokeColor := points[0]
	gc.SetFillColor(strokeColor.Color)
	gc.SetStrokeColor(strokeColor.Color)
	if len(points) == 1{
		s := points[0]
		gc.SetLineWidth(0)
		gc.ArcTo(float64(s.Point.X),
			float64(s.Point.Y),
			float64(s.Radius),
			float64(s.Radius),
			0, 2*math.Pi)	
		gc.FillStroke()
		return
	}
	x0, y0 := points[0].Point.X, points[0].Point.Y
	x1, y1 := points[1].Point.X, points[1].Point.Y
	for i := 1; i < len(points); i++{
		gc.QuadCurveTo(float64(x0), float64(y0), float64(x1), float64(y1))
		x0, y0 = x1, y1
		x1, y1 = points[i].Point.X, points[i].Point.Y
	}
}

func createCurve(cnv *image.RGBA,
	refImage image.Image,
	gradMag [][]float64,
	gradOri [][]float64,
	x0, y0, radius int,
	settings *PainterlySettings) []MyStroke{
	// ------
	MaxStrokeLength := settings.Style.MaximumStroke
	MinStrokeLength := settings.Style.MinimumStroke
	fc := settings.Style.FC
	
	strokeColor := refImage.At(x0, y0)
	output := []MyStroke{
		MyStroke{
			Color: RandomizeColor(strokeColor, settings),
			Point: image.Point{x0, y0},
			Radius: radius,
		},
	}
	x,y := x0, y0
	lastDX, lastDY := 0.0, 0.0

	for i := 1; i <= MaxStrokeLength; i++{
		x = IntMax(0, IntMin(x, len(gradMag[0])-1))
		y = IntMax(0, IntMin(y, len(gradMag)-1))

		// If we have already painted the minimal
		// stroke length and the paint color
		// differs from the color goal
		refColor := refImage.At(x, y)
		cnvColor := cnv.At(x, y)
		if i > MinStrokeLength && 
				(ColorDistance(refColor, cnvColor) < 
						ColorDistance(refColor, strokeColor)){
			return output
		}
		
		// Detect vanishing gradient
		if gradMag[y][x] == 0{
			return output			
		}
		
		// get unit vector of gradient
		gy, gx := math.Sincos(gradOri[y][x])
		dx, dy := -gx, gy
		
		// If necesary, reverse direction
		if lastDX*dx+lastDY*dy<0{
			dx, dy = -dx, -dy
		}
		
		// filter the stroke direction
		dx, dy = fc*dx+(1-fc)*lastDX, fc*dy+(1-fc)*lastDY
		temproot := math.Sqrt(dx*dx+dy*dy)
		dx, dy = dx/temproot, dy/temproot
		x, y = int(float64(x+radius)*dx), int(float64(y+radius)*dy)
		lastDX, lastDY = dx, dy
		
		// add new stroke
		output = append(output, MyStroke{
				Color: strokeColor,
				Point: image.Point{x, y},
				Radius: radius,
			})
	}
	return output
}
