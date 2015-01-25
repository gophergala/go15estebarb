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
	Size         int64
}

func (m *Image) GenerateID()string{
		return (string)(m.Blobkey)+"_"+m.OwnerID
}

func GenID(blobkey, oid string) string{
	return blobkey+"_"+oid
}

func ImagesPOST(c appengine.Context,
	usr *user.User,
	blobinfo *blobstore.BlobInfo,
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

func Images_OfUser_GET(c appengine.Context, usr *user.User) ([]Image, []*datastore.Key ,error){
	q := datastore.NewQuery("Images").
		Filter("OwnerID =", usr.ID).
		Order("-CreationTime")
	var images []Image
	keys, err := q.GetAll(c, &images)
	return images, keys, err
}

func Images_GetOne(c appengine.Context, usr *user.User, blobkey string)(*Image, error){
	q := datastore.NewQuery("Images").
	Filter("OwnerID =", usr.ID).
	Filter("Blobkey =", blobkey)
	var images []Image
	_, err := q.GetAll(c, &images)
	if err != nil && len(images)>0{
		return nil, err
	}
	m := images[0]
	return &m, nil
}

func Images_UpdateStyle(c appengine.Context,
		usr *user.User,
		blobkey string,
		newstyle string) (* datastore.Key, error){
	// Retrieve key
	m, err := Images_GetOne(c, usr, blobkey)
	if err != nil{
		return nil, err
	}
	
	// Updates the value
	if m.Style == newstyle || m.OwnerID != usr.ID{
		return nil, nil
	}
	
	m.Style = newstyle
	key := datastore.NewKey(c, "Images", m.GenerateID(), 0, nil)
	return datastore.Put(c, key, &m)
}

func Images_Delete(c appengine.Context,
	usr *user.User,
	blobkey string) error{
	m, err := Images_GetOne(c, usr, blobkey)
	if err != nil{
		return err
	}
	if m.OwnerID != usr.ID{
		return nil
	}
	key := datastore.NewKey(c, "Images", GenID(blobkey, usr.ID), 0, nil)
	return datastore.Delete(c, key)
}
