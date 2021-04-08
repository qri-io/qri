package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/base/archive"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

// DatasetHandlers wraps a requests struct to interface with http.HandlerFunc
type DatasetHandlers struct {
	lib.DatasetMethods
	inst     *lib.Instance
	ReadOnly bool
}

// NewDatasetHandlers allocates a DatasetHandlers pointer
func NewDatasetHandlers(inst *lib.Instance, readOnly bool) *DatasetHandlers {
	h := DatasetHandlers{inst.Dataset(), inst, readOnly}
	return &h
}

// SaveHandler is a dataset save/update endpoint
func (h *DatasetHandlers) SaveHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !(r.Method == http.MethodPut || r.Method == http.MethodPost) {
			util.NotFoundHandler(w, r)
			return
		}
		params := &lib.SaveParams{}
		err := lib.UnmarshalParams(r, params)
		if err != nil {
			log.Debugw("unmarshal dataset save error", "err", err)
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		scriptOutput := &bytes.Buffer{}
		params.ScriptOutput = scriptOutput

		got, _, err := h.inst.Dispatch(r.Context(), "dataset.save", params)
		if err != nil {
			log.Debugw("save dataset error", "err", err)
			util.RespondWithError(w, err)
			return
		}
		res, ok := got.(*dataset.Dataset)
		if !ok {
			util.RespondWithDispatchTypeError(w, got)
			return
		}

		// Don't leak paths across the API, it's possible they contain absolute paths or tmp dirs.
		res.BodyPath = filepath.Base(res.BodyPath)

		resRef := reporef.DatasetRef{
			Peername:  res.Peername,
			Name:      res.Name,
			ProfileID: profile.IDB58DecodeOrEmpty(res.ProfileID),
			Path:      res.Path,
			Dataset:   res,
		}

		msg := scriptOutput.String()
		util.WriteMessageResponse(w, msg, resRef)
	}

}

// RemoveHandler is a a dataset delete endpoint
func (h *DatasetHandlers) RemoveHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !(r.Method == http.MethodDelete || r.Method == http.MethodPost) {
			util.NotFoundHandler(w, r)
			return
		}
		params := &lib.RemoveParams{}
		err := lib.UnmarshalParams(r, params)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		if params.Remote != "" {
			res, err := h.inst.Remote().Remove(r.Context(), &lib.PushParams{
				Ref:    params.Ref,
				Remote: params.Remote,
			})
			if err != nil {
				log.Error("deleting dataset from remote: %s", err.Error())
				util.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			util.WriteResponse(w, res)
			return
		}
		got, _, err := h.inst.Dispatch(r.Context(), "dataset.remove", params)
		if err != nil {
			log.Infof("error removing dataset: %s", err.Error())
			util.RespondWithError(w, err)
			return
		}
		res, ok := got.(*lib.RemoveResponse)
		if !ok {
			util.RespondWithDispatchTypeError(w, got)
		}
		util.WriteResponse(w, res)
	}
}

// GetHandler is a dataset single endpoint
func (h *DatasetHandlers) GetHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !(r.Method == http.MethodGet || r.Method == http.MethodPost) {
			util.NotFoundHandler(w, r)
			return
		}
		params := &lib.GetParams{}

		err := lib.UnmarshalParams(r, params)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		got, _, err := h.inst.Dispatch(r.Context(), "dataset.get", params)
		if err != nil {
			util.RespondWithError(w, err)
			return
		}
		res, ok := got.(*lib.GetResult)
		if !ok {
			util.RespondWithDispatchTypeError(w, got)
			return
		}
		h.replyWithGetResponse(w, r, params, res)
	}

}

// PeerListHandler is a dataset list endpoint
func (h *DatasetHandlers) PeerListHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			util.NotFoundHandler(w, r)
			return
		}
		p := lib.ListParamsFromRequest(r)
		p.OrderBy = "created"

		// TODO - cheap peerId detection
		profileID := r.URL.Path[len("/list/"):]
		if len(profileID) > 0 && profileID[:2] == "Qm" {
			// TODO - let's not ignore this error
			p.ProfileID, _ = profile.IDB58Decode(profileID)
		} else {
			ref, err := lib.DsRefFromPath(r.URL.Path[len("/list/"):])
			if err != nil {
				util.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			if !ref.IsPeerRef() {
				util.WriteErrResponse(w, http.StatusBadRequest, errors.New("request needs to be in the form '/list/[peername]'"))
				return
			}
			p.Peername = ref.Username
		}

		res, err := h.inst.Collection().List(r.Context(), &p)
		if err != nil {
			log.Infof("error listing peer's datasets: %s", err.Error())
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		if err := util.WritePageResponse(w, res, r, p.Page()); err != nil {
			log.Infof("error list datasests response: %s", err.Error())
		}
	}
}

// PullHandler is an endpoint for creating new datasets
func (h *DatasetHandlers) PullHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !(r.Method == http.MethodPost || r.Method == http.MethodPut) {
			util.NotFoundHandler(w, r)
			return
		}
		params := &lib.PullParams{}
		if err := lib.UnmarshalParams(r, params); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		got, _, err := h.inst.Dispatch(r.Context(), "dataset.pull", params)
		if err != nil {
			log.Infof("error pulling dataset: %s", err.Error())
			util.RespondWithError(w, err)
			return
		}
		res, ok := got.(*dataset.Dataset)
		if !ok {
			util.RespondWithDispatchTypeError(w, got)
		}

		ref := reporef.DatasetRef{
			Peername: res.Peername,
			Name:     res.Name,
			Path:     res.Path,
			Dataset:  res,
		}
		util.WriteResponse(w, ref)
	}
}

// UnpackHandler unpacks a zip file and sends it back as json
func (h *DatasetHandlers) UnpackHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			util.NotFoundHandler(w, r)
			return
		}
		postData, err := ioutil.ReadAll(r.Body)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		contents, err := archive.UnzipGetContents(postData)
		if err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		data, err := json.Marshal(contents)
		if err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		util.WriteResponse(w, json.RawMessage(data))
	}
}

func extensionToMimeType(ext string) string {
	switch ext {
	case ".csv":
		return "text/csv"
	case ".json":
		return "application/json"
	case ".yaml":
		return "application/x-yaml"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".zip":
		return "application/zip"
	case ".txt":
		return "text/plain"
	default:
		return ""
	}
}

// inlineScriptsToBytes consumes all open script files for dataset components
// other than the body, inlining file data to scriptBytes fields
func inlineScriptsToBytes(ds *dataset.Dataset) error {
	var err error
	if ds.Readme != nil && ds.Readme.ScriptFile() != nil {
		if ds.Readme.ScriptBytes, err = ioutil.ReadAll(ds.Readme.ScriptFile()); err != nil {
			return err
		}
	}

	if ds.Transform != nil && ds.Transform.ScriptFile() != nil {
		if ds.Transform.ScriptBytes, err = ioutil.ReadAll(ds.Transform.ScriptFile()); err != nil {
			return err
		}
	}

	if ds.Viz != nil && ds.Viz.ScriptFile() != nil {
		if ds.Viz.ScriptBytes, err = ioutil.ReadAll(ds.Viz.ScriptFile()); err != nil {
			return err
		}
	}

	return nil
}

// replyWithGetResponse writes an http response back to the client, based upon what sort of
// response they requested. Handles raw file downloads (without response wrappers), zip downloads,
// body pagination, as well as normal head responses. Input logic has already been handled
// before this function, so errors should not commonly happen.
func (h *DatasetHandlers) replyWithGetResponse(w http.ResponseWriter, r *http.Request, params *lib.GetParams, result *lib.GetResult) {
	resultFormat := params.Format
	if resultFormat == "" {
		resultFormat = result.Dataset.Structure.Format
	}

	if resultFormat == "json" {
		// Convert components with scriptPaths (transform, readme, viz) in scriptBytes
		if err := inlineScriptsToBytes(result.Dataset); err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}

		if params.Selector != "" {
			page := util.PageFromRequest(r)
			dataResponse := lib.DataResponse{
				Path: result.Dataset.BodyPath,
				Data: json.RawMessage(result.Bytes),
			}
			stripServerSideQueryParams(r)
			if err := util.WritePageResponse(w, dataResponse, r, page); err != nil {
				log.Infof("error writing response: %s", err.Error())
			}
			return
		}

		// TODO (b5) - remove this. res.Ref should be used instead
		datasetRef := reporef.DatasetRef{
			Peername:  result.Dataset.Peername,
			ProfileID: profile.IDB58DecodeOrEmpty(result.Dataset.ProfileID),
			Name:      result.Dataset.Name,
			Path:      result.Dataset.Path,
			FSIPath:   result.FSIPath,
			Published: result.Published,
			Dataset:   result.Dataset,
		}

		util.WriteResponse(w, datasetRef)
	} else {
		filename, err := archive.GenerateFilename(result.Dataset, resultFormat)
		if err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", extensionToMimeType("."+resultFormat))
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		w.Write(result.Bytes)
	}
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
