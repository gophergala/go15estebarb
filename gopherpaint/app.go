package gopherpaint

import (
	"appengine"
	"appengine/blobstore"
	"appengine/memcache"
	"appengine/user"
	"bytes"
	"errors"
	"filters"
	"html/template"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"time"
)

var templates = map[string]*template.Template{
	"prepare": template.Must(template.ParseFiles("templates/prepare.html", "templates/scripts.html")),
	"home":    template.Must(template.ParseFiles("templates/home.html", "templates/scripts.html")),
	"share":   template.Must(template.ParseFiles("templates/share.html", "templates/scripts.html")),
}

func init() {
	http.HandleFunc("/delete", handleDelete)
	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/prepare", handleSetupPaint)
	http.HandleFunc("/render", handlePreview)
	http.HandleFunc("/share", handleShare)
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
	
	// if not logged in then fail
	u := user.Current(c)
	if u == nil{
		http.Redirect(w,r,"/", http.StatusFound)
	}

	file := blobs["file"]
	if len(file) == 0 {
		serveError(c, w, errors.New("no files uploaded"))
		return
	}
	ImagesPOST(c,u,file[0],"grayscale")
	http.Redirect(w, r, "/prepare?blobKey="+string(file[0].BlobKey), http.StatusFound)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	r.ParseForm()
	blobkey := r.FormValue("blobKey")
	usr := user.Current(c)
	if usr == nil{
		http.Redirect(w,r, "/", http.StatusFound)
		return
	}
	err := Images_Delete(c, usr, blobkey)
	if err != nil{
		serveError(c, w, err)
	}
	// Need for sleep. Or we aren't going to delete the image
	// before the next rendering of frontpage.
	time.Sleep(500 * time.Millisecond)
	http.Redirect(w,r, "/", http.StatusFound)
}

func handleShare(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	context := make(map[string]interface{})
	r.ParseForm()
	imgkey := r.FormValue("blobKey")
	context["imgkey"] = imgkey
	newstyle := r.FormValue("style")
	context["style"] = newstyle
	u := user.Current(c)
	if u != nil{
		_, err := Images_UpdateStyle(c, u, imgkey, newstyle)
		if err != nil {
			context["error"] = err
		}
	}
	
	templates["share"].Execute(w, context)
}

func handleSetupPaint(w http.ResponseWriter, r *http.Request) {
	//c := appengine.NewContext(r)
	context := make(map[string]interface{})
	context["imgkey"] = r.FormValue("blobKey")
	//context["ListOfStyles"] = ListOfStyles
	templates["prepare"].Execute(w, context)
}

func handler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	context := make(map[string]interface{})
	u := user.Current(c)
	var err error
	if u == nil {
		url, err := user.LoginURL(c, r.URL.String())
		if err != nil {
			serveError(c, w, err)
			return
		}
		context["IsLogged"] = false
		context["LoginURL"] = url
	} else {
		context["IsLogged"] = true
		context["UserName"] = u.String()
		context["LogoutURL"], err = user.LogoutURL(c, "/")
		if err != nil {
			serveError(c, w, err)
			return
		}
		pics, _, err := Images_OfUser_GET(c, u)
		if err != nil {
			serveError(c, w, err)
			return
		}
		context["Images"] = pics
	}
	uploadURL, err := blobstore.UploadURL(c, "/upload", nil)
	context["uploadURL"] = uploadURL.Path
	if err != nil {
		serveError(c, w, err)
	}
	w.Header().Set("Cache-Control", "private, no-store, max-age=0, no-cache, must-revalidate, post-check=0, pre-check=0")
	templates["home"].Execute(w, context)
}

func handlePreview(w http.ResponseWriter, r *http.Request) {
	size := 200
	switch r.FormValue("size") {
	case "800":
		size = 800
	default:
		size = 200
	}
	handleRender(w, r, size)
}

func handleRender(w http.ResponseWriter, r *http.Request, size int) {
	c := appengine.NewContext(r)
	r.ParseForm()
	blobkey := (appengine.BlobKey)(r.FormValue("blobKey"))
	attachment := r.FormValue("attachment")
	style := r.FormValue("style")

	if attachment == "1" {
		w.Header().Set("Content-Disposition", "attachment")
	}

	// First tries to retrieve it from memcache:
	item, err := memcache.Get(c, (string)(blobkey)+"_"+style+"_"+string(size))
	if err == nil {
		// Yay, we have the picture in cache
		w.Header().Set("Content-type", "image/png")
		w.Header().Set("Cache-control", "public, max-age=259200")
		w.Write(item.Value)
		return
	}

	rimg := blobstore.NewReader(c, blobkey)
	img, _, err := image.Decode(rimg)
	if err != nil {
		return
	}
	img = filters.RescaleImage(img, size)
	switch style {
	case "voronoi":
		img = filters.FilterVoronoi(c, img)
	case "oilpaint":
		img = filters.FilterOilPaint(c, img)
	case "impresionist":
		img = filters.FilterPainterlyStyles(c, img, &filters.PainterlySettings{
			Style:   filters.StyleImpressionist,
			Blobkey: blobkey,
		})
	case "expresionist":
		img = filters.FilterPainterlyStyles(c, img, &filters.PainterlySettings{
			Style:   filters.StyleExpressionist,
			Blobkey: blobkey,
		})
	case "coloristwash":
		img = filters.FilterPainterlyStyles(c, img, &filters.PainterlySettings{
			Style:   filters.StyleColoristWash,
			Blobkey: blobkey,
		})
	case "pointillist":
		img = filters.FilterPainterlyStyles(c, img, &filters.PainterlySettings{
			Style:   filters.StylePointillist,
			Blobkey: blobkey,
		})
	case "psychedelic":
		img = filters.FilterPainterlyStyles(c, img, &filters.PainterlySettings{
			Style:   filters.StylePsychedelic,
			Blobkey: blobkey,
		})
	default:
		style = "grayscale"
		img = filters.FilterGrayscale(c, img)
	}
	// Set the headers
	w.Header().Set("Content-type", "image/png")
	w.Header().Set("Cache-control", "public, max-age=259200")
	buffer := bytes.NewBuffer([]byte{})
	png.Encode(buffer, img)
	w.Write(buffer.Bytes())
	mcItem := &memcache.Item{
		Key:   (string)(blobkey) + "_" + style + "_" + string(size),
		Value: buffer.Bytes(),
	}
	memcache.Add(c, mcItem)
}
