package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/logging"
	"github.com/qri-io/qri/repo"
)

func NewDatasetHandlers(log logging.Logger, store cafs.Filestore, r repo.Repo) *DatasetHandlers {
	req := core.NewDatasetRequests(store, r)
	h := DatasetHandlers{*req, log, store}
	return &h
}

// DatasetHandlers wraps a requests struct to interface with http.HandlerFunc
type DatasetHandlers struct {
	core.DatasetRequests
	log   logging.Logger
	store cafs.Filestore
}

func (h *DatasetHandlers) DatasetsHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *DatasetHandlers) DatasetHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *DatasetHandlers) StructuredDataHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.getStructuredDataHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *DatasetHandlers) AddDatasetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST":
		h.addDatasetHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *DatasetHandlers) ZipDatasetHandler(w http.ResponseWriter, r *http.Request) {
	res := &dataset.Dataset{}
	args := &core.GetDatasetParams{
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

func (h *DatasetHandlers) listDatasetsHandler(w http.ResponseWriter, r *http.Request) {
	lp := core.ListParamsFromRequest(r)
	res := []*repo.DatasetRef{}
	args := &core.ListParams{
		Limit:   lp.Limit,
		Offset:  lp.Offset,
		OrderBy: "created", //TODO: should there be a global default orderby?
	}
	if err := h.List(args, &res); err != nil {
		h.log.Infof("error listing datasets: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	// TODO: need to update util.WritePageResponse to take a
	// core.ListParams rather than a util.Page struct
	// for time being I added an empty util.Page struct
	if err := util.WritePageResponse(w, res, r, util.Page{}); err != nil {
		h.log.Infof("error list datasests response: %s", err.Error())
	}
}

func (h *DatasetHandlers) getDatasetHandler(w http.ResponseWriter, r *http.Request) {
	res := &dataset.Dataset{}
	args := &core.GetDatasetParams{
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

func (h *DatasetHandlers) saveDatasetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Header.Get("Content-Type") {
	case "application/json":
		h.saveStructureHandler(w, r)
	default:
		h.initDatasetFileHandler(w, r)
	}
}

func (h *DatasetHandlers) updateDatasetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Header.Get("Content-Type") {
	case "application/json":
		h.updateMetadataHandler(w, r)
		// default:
		//  h.initDatasetFileHandler(w, r)
	}
}

func (h *DatasetHandlers) updateMetadataHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.Commit{}
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

func (h *DatasetHandlers) saveStructureHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.SaveParams{}
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

func (h *DatasetHandlers) initDatasetFileHandler(w http.ResponseWriter, r *http.Request) {
	infile, header, err := r.FormFile("file")
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	p := &core.InitDatasetParams{
		Name: r.FormValue("name"),
		Data: memfs.NewMemfileReader(header.Filename, infile),
	}
	res := &dataset.Dataset{}
	if err := h.InitDataset(p, res); err != nil {
		h.log.Infof("error initializing dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}

func (h *DatasetHandlers) deleteDatasetHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.DeleteParams{
		Name: r.FormValue("name"),
		Path: datastore.NewKey(r.URL.Path[len("/datasets"):]),
	}

	ds := &dataset.Dataset{}
	if err := h.Get(&core.GetDatasetParams{Name: p.Name, Path: p.Path}, ds); err != nil {
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

func (h *DatasetHandlers) getStructuredDataHandler(w http.ResponseWriter, r *http.Request) {
	//page := util.PageFromRequest(r)
	listParams := core.ListParamsFromRequest(r)

	all, err := util.ReqParamBool("all", r)
	if err != nil {
		all = false
	}

	objectRows, err := util.ReqParamBool("object_rows", r)
	if err != nil {
		objectRows = true
	}

	p := &core.StructuredDataParams{
		Format:  dataset.JsonDataFormat,
		Path:    datastore.NewKey(r.URL.Path[len("/data"):]),
		Objects: objectRows,
		Limit:   listParams.Limit,
		Offset:  listParams.Offset,
		All:     all,
	}
	data := &core.StructuredData{}
	if err := h.StructuredData(p, data); err != nil {
		h.log.Infof("error reading structured data: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, data)
}

func (h *DatasetHandlers) addDatasetHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.AddParams{
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
