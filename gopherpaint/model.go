package gopherpaint

import (
	"appengine"
	"appengine/datastore"
	"appengine/blobstore"
	"appengine/user"
	"time"
)

type Image struct {
	OwnerID      string
	Blobkey      appengine.BlobKey
	Style        string
	CreationTime time.Time
	MD5          string
	Size         int
}

func (m *Image) GenerateID()string{
		return m.MD5+"_"+m.OwnerID
}

func ImagesPOST(c appengine.Context,
	usr user.User,
	blobinfo blobstore.BlobInfo,
	style string) (* datastore.Key, error){
	data := &Image{
		OwnerID:      usr.ID,
		Blobkey:      blobinfo.BlobKey,
		Style:        style,
		CreationTime: time.Now(),
		MD5:          blobinfo.MD5,
		Size:         blobinfo.Size,
	}
	key := datastore.NewKey(c, "Images", data.GenerateID(), 0, nil)
	return datastore.Put(c, key, data)
}

func Images_OfUser_GET(c appengine.Context, usr user.User) ([]Image, []*datastore.Key ,err){
	q := datastore.NewQuery("Images").
		Filter("OwnerID =", usr.ID).
		Order("-CreationTime")
	var images []Image
	keys, err := q.GetAll(c, &images)
	return images, keys, err
}

func Images_UpdateStyle(c appengine.Context,
		usr user.User,
		blobkey string,
		newstyle string) (* datastore.Key, error){
	// Retrieve key
	q := datastore.NewQuery("Images").
		Filter("OwnerID =", usr.ID).
		Filter("BlobKey =", appengine.BlobKey(blobkey))
	t:=q.Run(c)
	var m Image
	_, err := t.Next(m)
	if err != nil{
		return nil, err
	}
	
	// Updates the value
	m.Style = newstyle
	key := datastore.NewKey(c, "Images", m.GenerateID(), 0, nil)
	return datastore.Put(c, key, m)
}
