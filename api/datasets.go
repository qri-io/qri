package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	util "github.com/datatogether/api/apiutil"
	// "github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
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

// PeerListHandler is a dataset list endpoint
func (h *DatasetHandlers) PeerListHandler(w http.ResponseWriter, r *http.Request) {
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
	args, err := DatasetRefFromPath(r.URL.Path[len("/export/"):])
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
	if args.Name == "" {
		util.WriteErrResponse(w, http.StatusBadRequest, errors.New("no dataset name or hash given"))
		return
	}
	err = h.Get(&args, res)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}

func (h *DatasetHandlers) peerListHandler(w http.ResponseWriter, r *http.Request) {
	p := core.ListParamsFromRequest(r)
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
	p.OrderBy = "created"
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
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		if p.DataFilename == "" {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("body of request must have 'datafilename' field"))
			return
		}
		if p.Data == nil {
			if !filepath.IsAbs(p.DataFilename) {
				util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("need absolute filepath"))
				return
			}
			data, err := os.Open(p.DataFilename)
			if err != nil {
				util.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			p.Data = data
			p.DataFilename = filepath.Base(p.DataFilename)
		}

		if p.Metadata == nil && p.MetadataFilename != "" {
			if !filepath.IsAbs(p.MetadataFilename) {
				util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("need absolute filepath for metadata"))
				return
			}
			metadata, err := os.Open(p.MetadataFilename)
			if err != nil {
				util.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			p.Metadata = metadata
			p.MetadataFilename = filepath.Base(p.MetadataFilename)
		}

		if p.Structure == nil && p.StructureFilename != "" {
			if !filepath.IsAbs(p.StructureFilename) {
				util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("need absolute filepath for structure"))
				return
			}
			structure, err := os.Open(p.StructureFilename)
			if err != nil {
				util.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			p.Structure = structure
			p.StructureFilename = filepath.Base(p.StructureFilename)
		}
	default:
		var f, m, s cafs.File
		infile, fileHeader, err := r.FormFile("file")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		metafile, metaHeader, err := r.FormFile("meta")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		structurefile, structureHeader, err := r.FormFile("structure")
		if err != nil && err != http.ErrMissingFile {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		f = memfs.NewMemfileReader(fileHeader.Filename, infile)
		m = memfs.NewMemfileReader(metaHeader.Filename, metafile)
		s = memfs.NewMemfileReader(structureHeader.Filename, structurefile)
		p = &core.InitParams{
			URL:               r.FormValue("url"),
			Name:              r.FormValue("name"),
			DataFilename:      fileHeader.Filename,
			Data:              f,
			MetadataFilename:  metaHeader.Filename,
			Metadata:          m,
			StructureFilename: structureHeader.Filename,
			Structure:         s,
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
	ref, err := DatasetRefFromPath(r.URL.Path[len("/add/"):])
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

// SaveReqParams is an encoding struct
// its intent is to be a more user-friendly structure for the api endpoint
// that will map to and from the core.SaveParams struct
type SaveReqParams struct {
	Name        string // the name of the dataset. required.
	SaveTitle   string // title of save message. required.
	SaveMessage string // details about changes made. optional.

	// At least one of DataFilename, StructureFilename, and/or MetaFilename required
	DataFilename      string // filename for new data.
	StructureFilename string // filename for new structure.
	MetaFileName      string // filename for new metadata
}

func (h *DatasetHandlers) saveHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Header.Get("Content-Type") {
	case "application/json":
		s := &SaveReqParams{}
		if err := json.NewDecoder(r.Body).Decode(s); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		if s.Name == "" || s.SaveTitle == "" {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("'name' and 'savetitle' required"))
			return
		}
		if s.DataFilename == "" && s.StructureFilename == "" && s.MetaFileName == "" {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("'datafilename' or 'structurefilename' or 'metafilename' required"))
			return
		}
		prevReq := &repo.DatasetRef{
			Name: s.Name,
		}
		prev := repo.DatasetRef{}
		if err := h.Get(prevReq, &prev); err != nil {
			util.WriteErrResponse(w, http.StatusNotFound, fmt.Errorf("error finding dataset to update: %s", err.Error()))
			return
		}
		save := &core.SaveParams{
			Prev: prev,
			Changes: &dataset.Dataset{
				Commit: &dataset.Commit{
					Title:   s.SaveTitle,
					Message: s.SaveMessage,
				},
			},
		}

		if s.MetaFileName != "" {
			metaFile, err := loadFileIfPath(s.MetaFileName)
			if err != nil {
				util.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			meta := &dataset.Meta{}
			err = json.NewDecoder(metaFile).Decode(meta)
			if err != nil {
				util.WriteErrResponse(w, http.StatusInternalServerError, err)
				return
			}
			save.Changes.Meta = meta
		}
		if s.StructureFilename != "" {
			stFile, err := loadFileIfPath(s.StructureFilename)
			if err != nil {
				util.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			st := &dataset.Structure{}
			err = json.NewDecoder(stFile).Decode(st)
			if err != nil {
				util.WriteErrResponse(w, http.StatusInternalServerError, err)
				return
			}
			save.Changes.Structure = st
		}
		if s.DataFilename != "" {
			dataFile, err := loadFileIfPath(s.DataFilename)
			if err != nil {
				util.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			if dataFile != nil {
				save.DataFilename = filepath.Base(s.DataFilename)
				save.Data = dataFile
			}
		}
		res := &repo.DatasetRef{}
		err := h.Save(save, res)
		if err != nil {
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
	p, err := DatasetRefFromPath(r.URL.Path[len("/remove/"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	ref := &repo.DatasetRef{}
	if err := h.Get(&repo.DatasetRef{Name: p.Name, Path: p.Path}, ref); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res := false
	if err := h.Remove(&p, &res); err != nil {
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
		current, err := repo.ParseDatasetRef(r.URL.Query().Get("current"))
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error parsing current param: %s", err.Error()))
			return
		}
		n, err := repo.ParseDatasetRef(r.URL.Query().Get("new"))
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error parsing new param: %s", err.Error()))
			return
		}
		p = &core.RenameParams{
			Current: current,
			New:     n,
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

func loadFileIfPath(path string) (file *os.File, err error) {
	if path == "" {
		return nil, nil
	}

	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("filepath must be absolute")
	}

	return os.Open(path)
}
