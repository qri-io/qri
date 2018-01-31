package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	util "github.com/datatogether/api/apiutil"
	// "github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/logging"
	"github.com/qri-io/qri/repo"
)

// DatasetHandlers wraps a requests struct to interface with http.HandlerFunc
type DatasetHandlers struct {
	core.DatasetRequests
	log  logging.Logger
	repo repo.Repo
}

// NewDatasetHandlers allocates a DatasetHandlers pointer
func NewDatasetHandlers(log logging.Logger, r repo.Repo) *DatasetHandlers {
	req := core.NewDatasetRequests(r, nil)
	h := DatasetHandlers{*req, log, r}
	return &h
}

// ListHandler is a dataset list endpoint
func (h *DatasetHandlers) ListHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.listHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// SaveHandler is a dataset save/update endpoint
func (h *DatasetHandlers) SaveHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "PUT", "POST":
		h.saveHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// RemoveHandler is a a dataset delete endpoint
func (h *DatasetHandlers) RemoveHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "DELETE", "POST":
		h.removeHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// GetHandler is a dataset single endpoint
func (h *DatasetHandlers) GetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.getHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// AddHandler is an endpoint for creating new datasets
func (h *DatasetHandlers) AddHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST", "PUT":
		h.addHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// RenameHandler is the endpoint for renaming datasets
func (h *DatasetHandlers) RenameHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST", "PUT":
		h.renameHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// ZipDatasetHandler is the endpoint for getting a zip archive of a dataset
func (h *DatasetHandlers) ZipDatasetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.zipDatasetHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *DatasetHandlers) zipDatasetHandler(w http.ResponseWriter, r *http.Request) {
	res := &repo.DatasetRef{}
	args := &repo.DatasetRef{
		Path: r.URL.Path[len("/export/"):],
		// Hash: r.FormValue("hash"),
	}
	err := h.Get(args, res)
	if err != nil {
		h.log.Infof("error getting dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("filename=\"%s.zip\"", "dataset"))
	dsutil.WriteZipArchive(h.repo.Store(), res.Dataset, w)
}

func (h *DatasetHandlers) listHandler(w http.ResponseWriter, r *http.Request) {
	args := core.ListParamsFromRequest(r)
	args.OrderBy = "created"
	res := []*repo.DatasetRef{}
	if err := h.List(&args, &res); err != nil {
		h.log.Infof("error listing datasets: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := util.WritePageResponse(w, res, r, args.Page()); err != nil {
		h.log.Infof("error list datasests response: %s", err.Error())
	}
}

func (h *DatasetHandlers) getHandler(w http.ResponseWriter, r *http.Request) {
	res := &repo.DatasetRef{}
	args, err := repo.ParseDatasetRef(r.URL.Path)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	if args.Path == "" {
		util.WriteErrResponse(w, http.StatusBadRequest, errors.New("no dataset name or hash given"))
		return
	}
	err = h.Get(args, res)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}

func (h *DatasetHandlers) addHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.InitParams{}
	switch r.Header.Get("Content-Type") {
	case "application/json":
		json.NewDecoder(r.Body).Decode(p)
	default:
		var f cafs.File
		infile, header, err := r.FormFile("file")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		f = memfs.NewMemfileReader(header.Filename, infile)
		p = &core.InitParams{
			URL:          r.FormValue("url"),
			Name:         r.FormValue("name"),
			DataFilename: header.Filename,
			Data:         f,
		}
	}

	res := &repo.DatasetRef{}
	if err := h.Init(p, res); err != nil {
		h.log.Infof("error initializing dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res.Dataset)
}

func (h *DatasetHandlers) saveHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Header.Get("Content-Type") {
	case "application/json":
		p := &core.SaveParams{}
		if err := json.NewDecoder(r.Body).Decode(p); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		res := &repo.DatasetRef{}
		if err := h.Save(p, res); err != nil {
			h.log.Infof("error updating dataset: %s", err.Error())
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		util.WriteResponse(w, res)
	default:
		util.WriteErrResponse(w, http.StatusBadRequest, errors.New("Content-Type of request body must be json"))
		return
	}
}

func (h *DatasetHandlers) removeHandler(w http.ResponseWriter, r *http.Request) {
	// TODO - use new fancy context handler?
	p := &repo.DatasetRef{
		Name: r.FormValue("name"),
		Path: r.URL.Path[len("/remove"):],
	}

	ref := &repo.DatasetRef{}
	if err := h.Get(&repo.DatasetRef{Name: p.Name, Path: p.Path}, ref); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res := false
	if err := h.Remove(p, &res); err != nil {
		h.log.Infof("error deleting dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, ref.Dataset)
}

func (h DatasetHandlers) renameHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.RenameParams{}
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(p); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
	} else {
		p = &core.RenameParams{
			Current: r.URL.Query().Get("current"),
			New:     r.URL.Query().Get("new"),
		}
	}

	res := &repo.DatasetRef{}
	if err := h.Rename(p, res); err != nil {
		h.log.Infof("error renaming dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	util.WriteResponse(w, res)
}
