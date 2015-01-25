package gopherpaint

import (
	"appengine"
	"appengine/blobstore"
	"appengine/delay"
	"filters"
	"html/template"
	"io"
	"net/http"
)

var processImageGrayscale = delay.Func("grayscale", filters.DoProcessingGray)
var processImageVoronoi = delay.Func("voronoi", filters.DoProcessingVoronoi)
var processImageOilPaint = delay.Func("oilpaint", filters.DoProcessingOilPaint)
var processImagePainterly = delay.Func("painterly", filters.DoProcessingPainterly)
var processImageMultiPainterly = delay.Func("multipaint", filters.DoProcessingMultiPainterly)


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
	//processImageGrayscale.Call(c, file[0].BlobKey)
	//processImageVoronoi.Call(c, file[0].BlobKey)
	//processImageOilPaint.Call(c, file[0].BlobKey)
	//processImagePainterly.Call(c, file[0].BlobKey)
	// We are going to create several paintings:
	processImageMultiPainterly.Call(c, &filters.PainterlySettings{
			Style: filters.StyleImpressionist,
			Blobkey: file[0].BlobKey,
	})
	processImageMultiPainterly.Call(c, &filters.PainterlySettings{
		Style: filters.StyleExpressionist,
		Blobkey: file[0].BlobKey,
	})
	processImageMultiPainterly.Call(c, &filters.PainterlySettings{
		Style: filters.StyleColoristWash,
		Blobkey: file[0].BlobKey,
	})
	processImageMultiPainterly.Call(c, &filters.PainterlySettings{
		Style: filters.StylePointillist,
		Blobkey: file[0].BlobKey,
	})
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
