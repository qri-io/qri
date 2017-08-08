package datasets

import (
	util "github.com/datatogether/api/apiutil"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/castore"
	"github.com/qri-io/dataset"
	"net/http"
)

func NewHandlers(store castore.Datastore, ns map[string]datastore.Key) *Handlers {
	r := NewRequests(store, ns)
	h := Handlers{*r}
	return &h
}

// Handlers wraps a requests struct to interface with http.HandlerFunc
type Handlers struct {
	Requests
}

func (d *Handlers) ListHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		d.listDatasetsHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (d *Handlers) GetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		d.getDatasetHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (d *Handlers) getDatasetHandler(w http.ResponseWriter, r *http.Request) {
	res := &dataset.Dataset{}
	args := &GetParams{
		Path: r.URL.Path[len("/datasets/"):],
		Hash: r.FormValue("hash"),
	}
	err := d.Get(args, res)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}

func (d *Handlers) listDatasetsHandler(w http.ResponseWriter, r *http.Request) {
	p := util.PageFromRequest(r)
	res := map[string]datastore.Key{}
	args := &ListParams{
		Limit:   p.Limit(),
		Offset:  p.Offset(),
		OrderBy: "created",
	}
	err := d.List(args, &res)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WritePageResponse(w, res, r, p)
}
