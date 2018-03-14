package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/dsdiff"
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

// DiffHandler is a dataset single endpoint
func (h *DatasetHandlers) DiffHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST", "GET":
		h.diffHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// PeerListHandler is a dataset list endpoint
func (h *DatasetHandlers) PeerListHandler(w http.ResponseWriter, r *http.Request) {
	h.log.Infof("%s %s", r.Method, r.URL.Path)
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.peerListHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// InitHandler is an endpoint for creating new datasets
func (h *DatasetHandlers) InitHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST", "PUT":
		h.initHandler(w, r)
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

// DataHandler gets a dataset's data
func (h *DatasetHandlers) DataHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.dataHandler(w, r)
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
	args, err := DatasetRefFromPath(r.URL.Path[len("/export"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	res := &repo.DatasetRef{}
	err = h.Get(&args, res)
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

	res := []repo.DatasetRef{}
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
	args, err := DatasetRefFromPath(r.URL.Path)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	if err = repo.CanonicalizeDatasetRef(h.repo, &args); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	err = h.Get(&args, res)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}

type diffAPIParams struct {
	Left, Right string
	Format      string
}

func (h *DatasetHandlers) diffHandler(w http.ResponseWriter, r *http.Request) {
	d := &diffAPIParams{}
	switch r.Header.Get("Content-Type") {
	case "application/json":
		if err := json.NewDecoder(r.Body).Decode(d); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error decoding body into params: %s", err.Error()))
			return
		}
	default:
		d.Left = r.FormValue("left")
		d.Right = r.FormValue("right")
		d.Format = r.FormValue("format")
	}

	leftReq, err := DatasetRefFromPath(d.Left)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error getting datasetRef from left path: %s", err.Error()))
		return
	}

	rightReq, err := DatasetRefFromPath(d.Right)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error getting datasetRef from right path: %s", err.Error()))
		return
	}

	left := &repo.DatasetRef{}
	if err = h.Get(&leftReq, left); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error getting left dataset from repo: %s", err))
		return
	}
	right := &repo.DatasetRef{}
	if err = h.Get(&rightReq, right); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error getting right dataset from repo: %s", err))
		return
	}

	diffs := make(map[string]*dsdiff.SubDiff)

	p := &core.DiffParams{
		DsLeft:  left.Dataset,
		DsRight: right.Dataset,
		DiffAll: true,
	}

	if err = h.Diff(p, &diffs); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error diffing datasets: %s", err))
	}

	if d.Format != "" {
		formattedDiffs, err := dsdiff.MapDiffsToString(diffs, d.Format)
		if err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error formating diffs: %s", err))
		}
		util.WriteResponse(w, formattedDiffs)
		return
	}

	util.WriteResponse(w, diffs)
}

func (h *DatasetHandlers) peerListHandler(w http.ResponseWriter, r *http.Request) {
	h.log.Info(r.URL.Path)
	p := core.ListParamsFromRequest(r)
	p.OrderBy = "created"

	// TODO - cheap peerId detection
	peerID := r.URL.Path[len("/list/"):]
	if len(peerID) > 0 && peerID[:2] == "Qm" {
		p.PeerID = peerID
	} else {
		ref, err := DatasetRefFromPath(r.URL.Path[len("/list/"):])
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		if !ref.IsPeerRef() {
			util.WriteErrResponse(w, http.StatusBadRequest, errors.New("request needs to be in the form '/list/[peername]'"))
			return
		}
		p.Peername = ref.Peername
	}

	res := []repo.DatasetRef{}
	if err := h.List(&p, &res); err != nil {
		h.log.Infof("error listing peer's datasets: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := util.WritePageResponse(w, res, r, p.Page()); err != nil {
		h.log.Infof("error list datasests response: %s", err.Error())
	}
}

func (h *DatasetHandlers) initHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.InitParams{}
	switch r.Header.Get("Content-Type") {
	case "application/json":
		if err := json.NewDecoder(r.Body).Decode(p); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error decoding body into params: %s", err.Error()))
			return
		}

		if p.URL == "" {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("if adding dataset using json, body of request must have 'url' field"))
			return
		}

	default:
		p = &core.InitParams{
			Peername: r.FormValue("peername"),
			URL:      r.FormValue("url"),
			Name:     r.FormValue("name"),
			Private:  r.FormValue("private") == "true",
		}

		infile, fileHeader, err := r.FormFile("file")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error opening data file: %s", err))
			return
		}
		if infile != nil {
			p.Data = cafs.NewMemfileReader(fileHeader.Filename, infile)
			p.DataFilename = fileHeader.Filename
		}

		metadatafile, metadataHeader, err := r.FormFile("metadata")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error opening metatdata file: %s", err))
			return
		}
		if metadatafile != nil {
			p.Metadata = cafs.NewMemfileReader(metadataHeader.Filename, metadatafile)
			p.MetadataFilename = metadataHeader.Filename
		}

		structurefile, structureHeader, err := r.FormFile("structure")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error opening structure file: %s", err))
			return
		}
		if structurefile != nil {
			p.Structure = cafs.NewMemfileReader(structureHeader.Filename, structurefile)
			p.StructureFilename = structureHeader.Filename
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

func (h *DatasetHandlers) addHandler(w http.ResponseWriter, r *http.Request) {
	ref, err := DatasetRefFromPath(r.URL.Path[len("/add"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	if ref.Peername == "" || ref.Name == "" {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("need peername and dataset name: '/add/[peername]/[datasetname]'"))
		return
	}

	res := repo.DatasetRef{}
	err = h.Add(&ref, &res)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

type saveParamsJSON struct {
	Peername  string          `json:"peername,omitempty"`
	Name      string          `json:"name,omitempty"`
	Title     string          `json:"title,omitempty"`
	Message   string          `json:"message,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Meta      json.RawMessage `json:"meta,omitempty"`
	Structure json.RawMessage `json:"structure,omitempty"`
}

func (h *DatasetHandlers) saveHandler(w http.ResponseWriter, r *http.Request) {
	save := &core.SaveParams{}
	if r.Header.Get("Content-Type") == "application/json" {
		saveParams := &saveParamsJSON{}
		err := json.NewDecoder(r.Body).Decode(saveParams)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		if strings.Contains(r.URL.Path, "/save/") {
			args, err := DatasetRefFromPath(r.URL.Path[len("/save/"):])
			if err != nil {
				util.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			if args.Peername != "" {
				saveParams.Peername = args.Peername
				saveParams.Name = args.Name
			}
		}

		save = &core.SaveParams{
			Peername: saveParams.Peername,
			Name:     saveParams.Name,
			Title:    saveParams.Title,
			Message:  saveParams.Message,
		}
		if len(saveParams.Data) != 0 {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("cannot accept data files using Content-Type: application/json. must make a mime/multipart request"))
			return
		}
		//  TODO - restore when we are sure we can accept json data with no errors
		// if len(saveParams.Data) != 0 {
		// 	save.Data = cafs.NewMemfileReader("data.json", bytes.NewReader(saveParams.Data))
		// 	save.DataFilename = "data.json"
		// }
		if len(saveParams.Meta) != 0 {
			save.Metadata = cafs.NewMemfileReader("meta.json", bytes.NewReader(saveParams.Meta))
			save.MetadataFilename = "meta.json"
		}
		if len(saveParams.Structure) != 0 {
			save.Structure = cafs.NewMemfileReader("structure.json", bytes.NewReader(saveParams.Structure))
			save.StructureFilename = "structure.json"
		}
	} else {
		save = &core.SaveParams{
			Peername: r.FormValue("peername"),
			URL:      r.FormValue("url"),
			Name:     r.FormValue("name"),
			Title:    r.FormValue("title"),
			Message:  r.FormValue("message"),
		}

		infile, fileHeader, err := r.FormFile("file")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error opening data file: %s", err))
			return
		}
		if infile != nil {
			save.Data = cafs.NewMemfileReader(fileHeader.Filename, infile)
			save.DataFilename = fileHeader.Filename
		}

		metadatafile, metadataHeader, err := r.FormFile("metadata")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error opening metatdata file: %s", err))
			return
		}
		if metadatafile != nil {
			save.Metadata = cafs.NewMemfileReader(metadataHeader.Filename, metadatafile)
			save.MetadataFilename = metadataHeader.Filename
		}

		structurefile, structureHeader, err := r.FormFile("structure")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error opening structure file: %s", err))
			return
		}
		if structurefile != nil {
			save.Structure = cafs.NewMemfileReader(structureHeader.Filename, structurefile)
			save.StructureFilename = structureHeader.Filename
		}
	}

	res := &repo.DatasetRef{}
	if err := h.Save(save, res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}

func (h *DatasetHandlers) removeHandler(w http.ResponseWriter, r *http.Request) {
	p, err := DatasetRefFromPath(r.URL.Path[len("/remove"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	ref := &repo.DatasetRef{}
	if err := h.Get(&p, ref); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res := false
	if err := h.Remove(ref, &res); err != nil {
		h.log.Infof("error deleting dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, ref)
}

// RenameReqParams is an encoding struct
// its intent is to be a more user-friendly structure for the api endpoint
// that will map to and from the core.RenameParams struct
type RenameReqParams struct {
	Current string
	New     string
}

func (h DatasetHandlers) renameHandler(w http.ResponseWriter, r *http.Request) {
	reqParams := &RenameReqParams{}
	p := &core.RenameParams{}
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(reqParams); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
	} else {
		reqParams.Current = r.URL.Query().Get("current")
		reqParams.New = r.URL.Query().Get("new")
	}
	current, err := repo.ParseDatasetRef(reqParams.Current)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error parsing current param: %s", err.Error()))
		return
	}
	n, err := repo.ParseDatasetRef(reqParams.New)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error parsing new param: %s", err.Error()))
		return
	}
	p = &core.RenameParams{
		Current: current,
		New:     n,
	}

	res := &repo.DatasetRef{}
	if err := h.Rename(p, res); err != nil {
		h.log.Infof("error renaming dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	util.WriteResponse(w, res)
}

func loadFileIfPath(path string) (file *os.File, err error) {
	if path == "" {
		return nil, nil
	}

	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("filepath must be absolute")
	}

	return os.Open(path)
}

// default number of entries to limit to when reading
// TODO - should move this into core
const defaultDataLimit = 100

// DataResponse is the struct used to respond to api requests made to the /data endpoint
// It is necessary because we need to include the 'path' field in the response
type DataResponse struct {
	Path string          `json:"path"`
	Data json.RawMessage `json:"data"`
}

func (h DatasetHandlers) dataHandler(w http.ResponseWriter, r *http.Request) {
	d, err := DatasetRefFromPath(r.URL.Path[len("/data"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	if err := repo.CanonicalizeDatasetRef(h.repo, &d); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
	}

	limit, err := util.ReqParamInt("limit", r)
	if err != nil {
		limit = defaultDataLimit
		err = nil
	}
	offset, err := util.ReqParamInt("offset", r)
	if err != nil {
		offset = 0
		err = nil
	}

	p := &core.StructuredDataParams{
		Path:   d.Path,
		Format: dataset.JSONDataFormat,
		Limit:  limit,
		Offset: offset,
		All:    r.FormValue("all") == "true" && limit == defaultDataLimit && offset == 0,
	}

	data := &core.StructuredData{}
	if err := h.StructuredData(p, data); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	page := util.PageFromRequest(r)
	dataResponse := DataResponse{
		Path: data.Path,
		Data: json.RawMessage(data.Data),
	}
	if err := util.WritePageResponse(w, dataResponse, r, page); err != nil {
		h.log.Infof("error writing repsonse: %s", err.Error())
	}
}
