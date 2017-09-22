package datasets

import (
	"bytes"
	"encoding/json"
	util "github.com/datatogether/api/apiutil"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/castore"
	"github.com/qri-io/qri/repo"
	"io/ioutil"
	"time"
	// "github.com/qri-io/castore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"net/http"
)

func NewHandlers(store castore.Datastore, r repo.Repo) *Handlers {
	req := NewRequests(store, r)
	h := Handlers{*req}
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
	res := []*dataset.DatasetRef{}
	args := &ListParams{
		Limit:   p.Limit(),
		Offset:  p.Offset(),
		OrderBy: "created",
	}
	if err := d.List(args, &res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WritePageResponse(w, res, r, p)
}

func (h *Handlers) getDatasetHandler(w http.ResponseWriter, r *http.Request) {
	res := &dataset.Dataset{}
	args := &GetParams{
		Path: datastore.NewKey(r.URL.Path[len("/datasets/"):]),
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
	switch r.Header.Get("Content-Type") {
	case "application/json":
		h.saveStructureHandler(w, r)
	default:
		h.initDatasetFileHandler(w, r)
	}
}

func (h *Handlers) saveStructureHandler(w http.ResponseWriter, r *http.Request) {
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

// TODO - move this into a request method
func (h *Handlers) initDatasetFileHandler(w http.ResponseWriter, r *http.Request) {
	infile, header, err := r.FormFile("file")
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	// TODO - split this into some sort of re-readable reader instead
	// of reading the entire file
	data, err := ioutil.ReadAll(infile)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	st, err := detect.FromReader(header.Filename, bytes.NewReader(data))
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	datakey, err := h.store.Put(data, true)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	adr := detect.Camelize(header.Filename)
	if r.FormValue("name") != "" {
		adr = detect.Camelize(r.FormValue("name"))
	}

	// ns, err := h.repo.Namespace()
	// if err != nil {
	// 	util.WriteErrResponse(w, http.StatusInternalServerError, err)
	// 	return
	// }

	ds := &dataset.Dataset{
		Timestamp: time.Now().In(time.UTC),
		Title:     adr,
		Data:      datakey,
		Structure: st,
	}

	dskey, err := ds.Save(h.store, true)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	if err = h.repo.PutDataset(dskey, ds); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	if err = h.repo.PutName(adr, dskey); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	// ns[adr] = dskey
	// if err := h.repo.SaveNamespace(ns); err != nil {
	// 	util.WriteErrResponse(w, http.StatusBadRequest, err)
	// 	return
	// }

	util.WriteResponse(w, ds)
}

func (h *Handlers) deleteDatasetHandler(w http.ResponseWriter, r *http.Request) {
	p := &DeleteParams{
		Name: r.FormValue("name"),
		Path: datastore.NewKey(r.URL.Path[len("/datasets"):]),
	}

	ds := &dataset.Dataset{}
	if err := h.Get(&GetParams{Name: p.Name, Path: p.Path}, ds); err != nil {
		return
	}

	res := false
	if err := h.Delete(p, &res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, ds)
}

func (h *Handlers) getStructuredDataHandler(w http.ResponseWriter, r *http.Request) {
	page := util.PageFromRequest(r)

	all, err := util.ReqParamBool("all", r)
	if err != nil {
		all = false
		// util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("invalid param 'all': %s", err.Error()))
		// return
	}

	p := &StructuredDataParams{
		Format: dataset.JsonDataFormat,
		Path:   datastore.NewKey(r.URL.Path[len("/data"):]),
		Limit:  page.Limit(),
		Offset: page.Offset(),
		All:    all,
	}
	data := &StructuredData{}
	if err := h.StructuredData(p, data); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, data)
}
