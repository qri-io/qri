package datasets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	util "github.com/datatogether/api/apiutil"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/server/logging"
)

func NewHandlers(log logging.Logger, store cafs.Filestore, r repo.Repo) *Handlers {
	req := NewRequests(store, r)
	h := Handlers{*req, log}
	return &h
}

// Handlers wraps a requests struct to interface with http.HandlerFunc
type Handlers struct {
	Requests
	log logging.Logger
}

func (h *Handlers) DatasetsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.listDatasetsHandler(w, r)
	case "PUT":
		h.updateDatasetHandler(w, r)
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
	case "POST":
		h.saveDatasetHandler(w, r)
	case "PUT":
		h.updateDatasetHandler(w, r)
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

func (h *Handlers) AddDatasetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST":
		h.addDatasetHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *Handlers) ZipDatasetHandler(w http.ResponseWriter, r *http.Request) {
	res := &dataset.Dataset{}
	args := &GetParams{
		Path: datastore.NewKey(r.URL.Path[len("/download/"):]),
		Hash: r.FormValue("hash"),
	}
	err := h.Get(args, res)
	if err != nil {
		h.log.Infof("error getting dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("filename=\"%s.zip\"", "dataset"))
	dsutil.WriteZipArchive(h.store, res, w)
}

func (h *Handlers) listDatasetsHandler(w http.ResponseWriter, r *http.Request) {
	p := util.PageFromRequest(r)
	res := []*repo.DatasetRef{}
	args := &ListParams{
		Limit:   p.Limit(),
		Offset:  p.Offset(),
		OrderBy: "created",
	}
	if err := h.List(args, &res); err != nil {
		h.log.Infof("error listing datasets: %s", err.Error())
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

func (h *Handlers) updateDatasetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Header.Get("Content-Type") {
	case "application/json":
		h.updateMetadataHandler(w, r)
		// default:
		// 	h.initDatasetFileHandler(w, r)
	}
}

func (h *Handlers) updateMetadataHandler(w http.ResponseWriter, r *http.Request) {
	p := &Commit{}
	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	res := &repo.DatasetRef{}
	if err := h.Update(p, res); err != nil {
		h.log.Infof("error updating dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}

func (h *Handlers) saveStructureHandler(w http.ResponseWriter, r *http.Request) {
	p := &SaveParams{}
	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	res := &dataset.Dataset{}
	if err := h.Save(p, res); err != nil {
		h.log.Infof("error saving dataset: %s", err.Error())
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

	datakey, err := h.store.Put(memfs.NewMemfileBytes("data."+st.Format.String(), data), true)
	if err != nil {
		h.log.Infof("error putting data file in store: %s", err.Error())
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

	dskey, err := dsfs.SaveDataset(h.store, ds, true)
	if err != nil {
		h.log.Infof("error saving dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	if err = h.repo.PutDataset(dskey, ds); err != nil {
		h.log.Infof("error putting dataset in repo: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	if err = h.repo.PutName(adr, dskey); err != nil {
		h.log.Infof("error adding dataset name to repo: %s", err.Error())
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
		h.log.Infof("error deleting dataset: %s", err.Error())
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
	}

	objectRows, err := util.ReqParamBool("object_rows", r)
	if err != nil {
		objectRows = true
	}

	p := &StructuredDataParams{
		Format:  dataset.JsonDataFormat,
		Path:    datastore.NewKey(r.URL.Path[len("/data"):]),
		Objects: objectRows,
		Limit:   page.Limit(),
		Offset:  page.Offset(),
		All:     all,
	}
	data := &StructuredData{}
	if err := h.StructuredData(p, data); err != nil {
		h.log.Infof("error reading structured data: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	if data, ok := data.Data.([]byte); ok {
		h.log.Info(string(data))
	}

	util.WriteResponse(w, data)
}

func (h *Handlers) addDatasetHandler(w http.ResponseWriter, r *http.Request) {
	p := &AddParams{
		Name: r.URL.Query().Get("name"),
		Hash: r.URL.Path[len("/add/"):],
	}

	res := &repo.DatasetRef{}
	if err := h.AddDataset(p, res); err != nil {
		h.log.Infof("error adding dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}
