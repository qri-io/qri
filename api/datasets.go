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
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

// DatasetHandlers wraps a requests struct to interface with http.HandlerFunc
type DatasetHandlers struct {
	lib.DatasetMethods
	inst     *lib.Instance
	remote   *lib.RemoteMethods
	ReadOnly bool
}

// NewDatasetHandlers allocates a DatasetHandlers pointer
func NewDatasetHandlers(inst *lib.Instance, readOnly bool) *DatasetHandlers {
	dsm := lib.NewDatasetMethods(inst)
	rm := lib.NewRemoteMethods(inst)
	h := DatasetHandlers{*dsm, inst, rm, readOnly}
	return &h
}

// ListHandler is a dataset list endpoint
func (h *DatasetHandlers) ListHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		if h.ReadOnly {
			readOnlyResponse(w, "/list")
			return
		}
		h.listHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// SaveHandler is a dataset save/update endpoint
func (h *DatasetHandlers) SaveHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut, http.MethodPost:
		h.saveHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// RemoveHandler is a a dataset delete endpoint
func (h *DatasetHandlers) RemoveHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodDelete, http.MethodPost:
		h.removeHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// GetHandler is a dataset single endpoint
func (h *DatasetHandlers) GetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		h.getHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// DiffHandler is a dataset single endpoint
func (h *DatasetHandlers) DiffHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost, http.MethodGet:
		if h.ReadOnly {
			readOnlyResponse(w, "/diff")
			return
		}
		h.diffHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// ChangesHandler is the endpoint for showing the changes between two datasets
func (h *DatasetHandlers) ChangesHandler(routePrefix string) http.HandlerFunc {
	handleChanges := h.changesHandler(routePrefix)

	return func(w http.ResponseWriter, r *http.Request) {
		if h.ReadOnly {
			readOnlyResponse(w, routePrefix)
			return
		}

		switch r.Method {
		default:
			util.NotFoundHandler(w, r)
		case http.MethodGet, http.MethodPost:
			handleChanges(w, r)
		}
	}
}

// PeerListHandler is a dataset list endpoint
func (h *DatasetHandlers) PeerListHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.peerListHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// PullHandler is an endpoint for creating new datasets
func (h *DatasetHandlers) PullHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost, http.MethodPut:
		h.pullHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// ValidateHandler is the endpoint for validating datasets
func (h *DatasetHandlers) ValidateHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		h.validateHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// RenameHandler is the endpoint for renaming datasets
func (h *DatasetHandlers) RenameHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost, http.MethodPut:
		h.renameHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// UnpackHandler unpacks a zip file and sends it back as json
func (h *DatasetHandlers) UnpackHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		postData, err := ioutil.ReadAll(r.Body)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		h.unpackHandler(w, r, postData)
	default:
		util.NotFoundHandler(w, r)
	}
}

// ManifestHandler is the endpoint for generating a manifest for a dataset path
func (h *DatasetHandlers) ManifestHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.manifestHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// ManifestMissingHandler is the endpoint for generating a manifest of blocks that are not present on this repo for a given manifest
func (h *DatasetHandlers) ManifestMissingHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.manifestMissingHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// DAGInfoHandler is the endpoint for generating a dag.Info for a dataset path
func (h *DatasetHandlers) DAGInfoHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.dagInfoHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
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

func (h *DatasetHandlers) listHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.ListParams{}
	if err := lib.UnmarshalParams(r, params); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	resRaw := ""
	res := []dsref.VersionInfo{}
	var err error
	if params.Raw {
		got, _, err := h.inst.Dispatch(r.Context(), "dataset.listrawrefs", params)
		if err != nil {
			util.RespondWithError(w, err)
			return
		}
		ok := false
		resRaw, ok = got.(string)
		if !ok {
			util.RespondWithDispatchTypeError(w, got)
			return
		}
	} else {
		got, _, err := h.inst.Dispatch(r.Context(), "dataset.list", params)
		if err != nil {
			util.RespondWithError(w, err)
			return
		}
		ok := false
		res, ok = got.([]dsref.VersionInfo)
		if !ok {
			util.RespondWithDispatchTypeError(w, got)
			return
		}
	}

	if err != nil {
		if errors.Is(err, lib.ErrListWarning) {
			log.Error(err)
			err = nil
		} else {
			log.Infof("error listing datasets: %s", err.Error())
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
	}

	if params.Raw {
		w.Header().Set("Content-Type", extensionToMimeType(".txt"))
		w.Write([]byte(resRaw))
		return
	}

	if err := util.WritePageResponse(w, res, r, params.Page()); err != nil {
		log.Infof("error list datasests response: %s", err.Error())
	}
}

func (h *DatasetHandlers) getHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *DatasetHandlers) diffHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.DiffParams{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.Diff(r.Context(), params)
	if err != nil {
		fmt.Println(err)
		util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error generating diff: %s", err.Error()))
		return
	}

	util.WritePageResponse(w, res, r, util.Page{})
}

func (h *DatasetHandlers) changesHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		method := "dataset.changereport"
		p := h.inst.NewInputParam(method)

		if err := lib.UnmarshalParams(r, p); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		res, _, err := h.inst.Dispatch(r.Context(), method, p)
		if err != nil {
			util.RespondWithError(w, err)
			return
		}
		util.WriteResponse(w, res)
		return
	}
}

func (h *DatasetHandlers) peerListHandler(w http.ResponseWriter, r *http.Request) {
	log.Info(r.URL.Path)
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

	res, err := h.List(r.Context(), &p)
	if err != nil {
		log.Infof("error listing peer's datasets: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := util.WritePageResponse(w, res, r, p.Page()); err != nil {
		log.Infof("error list datasests response: %s", err.Error())
	}
}

func (h *DatasetHandlers) pullHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.PullParams{}
	if err := lib.UnmarshalParams(r, params); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.Pull(r.Context(), params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	ref := reporef.DatasetRef{
		Peername: res.Peername,
		Name:     res.Name,
		Path:     res.Path,
		Dataset:  res,
	}
	util.WriteResponse(w, ref)
}

func (h *DatasetHandlers) saveHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *DatasetHandlers) removeHandler(w http.ResponseWriter, r *http.Request) {
	params := lib.RemoveParams{}
	err := lib.UnmarshalParams(r, &params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	if params.Remote != "" {
		res := &dsref.Ref{}
		err := h.remote.Remove(&lib.PushParams{
			Ref:        params.Ref,
			RemoteName: params.Remote,
		}, res)
		if err != nil {
			log.Error("deleting dataset from remote: %s", err.Error())
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		util.WriteResponse(w, res)
		return
	}

	res, err := h.Remove(r.Context(), &params)
	if err != nil {
		log.Infof("error deleting dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h DatasetHandlers) validateHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.ValidateParams{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.Validate(r.Context(), params)
	if err != nil {
		log.Infof("error validating dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h DatasetHandlers) renameHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.RenameParams{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.Rename(r.Context(), params)
	if err != nil {
		log.Infof("error renaming dataset: %s", err.Error())
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

func (h DatasetHandlers) unpackHandler(w http.ResponseWriter, r *http.Request, postData []byte) {
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

func (h DatasetHandlers) manifestHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.ManifestParams{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.Manifest(r.Context(), params)
	if err != nil {
		log.Infof("error generating manifest: %s", err.Error())
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h DatasetHandlers) manifestMissingHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.ManifestMissingParams{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.ManifestMissing(r.Context(), params)
	if err != nil {
		log.Infof("error generating manifest: %s", err.Error())
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h DatasetHandlers) dagInfoHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.DAGInfoParams{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.DAGInfo(r.Context(), params)
	if err != nil {
		log.Infof("error generating manifest: %s", err.Error())
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	util.WriteResponse(w, res)
}
