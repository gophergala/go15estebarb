package gopherpaint

import (
	"appengine"
	"appengine/blobstore"
	"appengine/delay"
	"filters"
	"html/template"
	"io"
	"net/http"
	"image"
	"image/png"
	_ "image/jpeg"
	_ "image/gif"
)

var processImageGrayscale = delay.Func("grayscale", filters.DoProcessingGray)
var processImageVoronoi = delay.Func("voronoi", filters.DoProcessingVoronoi)
var processImageOilPaint = delay.Func("oilpaint", filters.DoProcessingOilPaint)
var processImagePainterly = delay.Func("painterly", filters.DoProcessingPainterly)
var processImageMultiPainterly = delay.Func("multipaint", filters.DoProcessingMultiPainterly)


func init() {
	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/serve/", handleServe)
	http.HandleFunc("/preview", handlePreview)
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
	processImageVoronoi.Call(c, file[0].BlobKey)
	processImageOilPaint.Call(c, file[0].BlobKey)
	//processImagePainterly.Call(c, file[0].BlobKey)
	// We are going to create several paintings:
	//*
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
	//*/
	
	processImageMultiPainterly.Call(c, &filters.PainterlySettings{
		Style: filters.StylePointillist,
		Blobkey: file[0].BlobKey,
	})

	processImageMultiPainterly.Call(c, &filters.PainterlySettings{
		Style: filters.StylePsychedelic,
		Blobkey: file[0].BlobKey,
	})

	http.Redirect(w, r, "/serve/?blobKey="+string(file[0].BlobKey), http.StatusFound)
}

func handleServe(w http.ResponseWriter, r *http.Request) {
	blobstore.Send(w, appengine.BlobKey(r.FormValue("blobKey")))
}

func handlePreview(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	blobkey := (appengine.BlobKey)(r.FormValue("blobKey"))
	style := r.FormValue("style")
	rimg := blobstore.NewReader(c, blobkey)
	img, _, err := image.Decode(rimg)
	if err != nil{
		return
	}
	img = filters.RescaleImage(img, 300)
	switch style{
	case "voronoi":
		img = filters.FilterVoronoi(c, img)
	case "oilpaint":
		img = filters.FilterOilPaint(c, img)
	case "impresionist":
		img = filters.FilterPainterlyStyles(c, img, &filters.PainterlySettings{
			Style: filters.StyleImpressionist,
			Blobkey: blobkey,
		})
	case "expresionist":
		img = filters.FilterPainterlyStyles(c, img, &filters.PainterlySettings{
			Style: filters.StyleExpressionist,
			Blobkey: blobkey,
		})
	case "coloristwash":
		img = filters.FilterPainterlyStyles(c, img, &filters.PainterlySettings{
			Style: filters.StyleColoristWash,
			Blobkey: blobkey,
		})
	case "pointillist":
		img = filters.FilterPainterlyStyles(c, img, &filters.PainterlySettings{
			Style: filters.StylePointillist,
			Blobkey: blobkey,
		})
	case "psychedelic":
		img = filters.FilterPainterlyStyles(c, img, &filters.PainterlySettings{
			Style: filters.StylePsychedelic,
			Blobkey: blobkey,
		})
	default:
		img = filters.FilterGrayscale(c, img)
	}
	png.Encode(w, img)
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
