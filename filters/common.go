package filters

import (
	"appengine"
	"appengine/blobstore"
	"github.com/disintegration/imaging"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"math"
)

func DoProcessingGray(c appengine.Context, blobkey appengine.BlobKey) {
	r := blobstore.NewReader(c, blobkey)
	img, _, err := image.Decode(r)
	if err != nil {
		c.Errorf("%v", err)
	}
	img = rescaleImage(img)
	pic := FilterGrayscale(c, img)
	saveImage(c, pic)
}

func DoProcessingVoronoi(c appengine.Context, blobkey appengine.BlobKey) {
	r := blobstore.NewReader(c, blobkey)
	img, _, err := image.Decode(r)
	if err != nil {
		c.Errorf("%v", err)
	}
	img = rescaleImage(img)
	pic := FilterVoronoi(c, img)
	saveImage(c, pic)
}

func DoProcessingOilPaint(c appengine.Context, blobkey appengine.BlobKey) {
	r := blobstore.NewReader(c, blobkey)
	img, _, err := image.Decode(r)
	if err != nil {
		c.Errorf("%v", err)
	}
	img = rescaleImage(img)
	pic := FilterOilPaint(c, img)
	saveImage(c, pic)
}

func DoProcessingPainterly(c appengine.Context, blobkey appengine.BlobKey) {
	r := blobstore.NewReader(c, blobkey)
	img, _, err := image.Decode(r)
	if err != nil {
		c.Errorf("%v", err)
	}
	img = rescaleImage(img)
	pic := FilterPainterly(c, img)
	saveImage(c, pic)
}

func saveImage(c appengine.Context, m image.Image) {
	w, err := blobstore.Create(c, "image/png")
	if err != nil {
		c.Errorf("%v", err)
	}
	defer w.Close()
	png.Encode(w, m)
}

func rescaleImage(m image.Image) image.Image {
	bounds := m.Bounds()
	ys := bounds.Max.Y
	xs := bounds.Max.X
	if xs > ys {
		return imaging.Resize(m, IntMin(800, xs), 0, imaging.Lanczos)
	} else {
		return imaging.Resize(m, 0, IntMin(800, ys), imaging.Lanczos)
	}
}

func bidimensionalArray(x, y int) [][]int {
	res := make([][]int, x)
	for i := range res {
		res[i] = make([]int, y)
	}
	return res
}

func distance(A []int, x, y int) float64 {
	dy := A[1] - y
	dx := A[0] - x
	return math.Sqrt(float64(dy*dy + dx*dx))
}

func manhattan(A []int, x, y int) float64 {
	dy := A[1] - y
	dx := A[0] - x
	return math.Abs(float64(dy)) + math.Abs(float64(dx))
}

func colorMean(colors []color.Color) color.Color {
	var r, g, b, a float64
	r, g, b, a = 0, 0, 0, 0
	for _, v := range colors {
		R, G, B, A := v.RGBA()
		r += float64(R)
		g += float64(G)
		b += float64(B)
		a += float64(A)
	}
	c := float64(len(colors))
	return color.NRGBA{uint8(r / c), uint8(g / c), uint8(b / c), uint8(a / c)}
}

type MyColor struct {
	R, G, B, A, C int64
}

func (o *MyColor) Add(c color.Color) {
	r, g, b, a := c.RGBA()
	o.R += int64(r)
	o.G += int64(g)
	o.B += int64(b)
	o.A += int64(a)
	o.C++
}

func (o *MyColor) Average() color.Color {
	if o.C == 0 {
		return color.Black
	}
	return color.NRGBA64{R: uint16(o.R / o.C), G: uint16(o.G / o.C), B: uint16(o.B / o.C), A: uint16(o.A / o.C)}
}

func IntMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func IntMin(a, b int) int {
	if a > b {
		return b
	}
	return a
}
