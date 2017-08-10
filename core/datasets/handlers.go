package datasets

import (
	"encoding/json"
	util "github.com/datatogether/api/apiutil"
	"github.com/ipfs/go-datastore"
	// "github.com/qri-io/castore"
	"github.com/qri-io/castore/ipfs"
	"github.com/qri-io/dataset"
	"net/http"
)

func NewHandlers(store *ipfs_datastore.Datastore, ns map[string]datastore.Key, nspath string) *Handlers {
	r := NewRequests(store, ns, nspath)
	h := Handlers{*r}
	return &h
}

// Handlers wraps a requests struct to interface with http.HandlerFunc
type Handlers struct {
	Requests
}

func (h *Handlers) DatasetsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.listDatasetsHandler(w, r)
	case "POST":
		h.saveDatasetHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *Handlers) DatasetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.getDatasetHandler(w, r)
	case "PUT", "POST":
		h.saveDatasetHandler(w, r)
	case "DELETE":
		h.deleteDatasetHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *Handlers) StructuredDataHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.getStructuredDataHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (d *Handlers) listDatasetsHandler(w http.ResponseWriter, r *http.Request) {
	p := util.PageFromRequest(r)
	res := []*dataset.Dataset{}
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

func (h *Handlers) getDatasetHandler(w http.ResponseWriter, r *http.Request) {
	res := &dataset.Dataset{}
	args := &GetParams{
		Path: r.URL.Path[len("/datasets/"):],
		Hash: r.FormValue("hash"),
	}
	err := h.Get(args, res)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}

func (h *Handlers) saveDatasetHandler(w http.ResponseWriter, r *http.Request) {
	p := &SaveParams{}
	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	res := &dataset.Dataset{}
	if err := h.Save(p, res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h *Handlers) deleteDatasetHandler(w http.ResponseWriter, r *http.Request) {
	p := &DeleteParams{}
	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	res := false
	if err := h.Delete(p, &res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h *Handlers) getStructuredDataHandler(w http.ResponseWriter, r *http.Request) {
	p := &StructuredDataParams{
		Path: datastore.NewKey(r.URL.Path[len("/data"):]),
	}
	data := &StructuredData{}
	if err := h.StructuredData(p, data); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, data)
}
