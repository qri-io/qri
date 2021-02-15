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
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/base/archive"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

// DatasetHandlers wraps a requests struct to interface with http.HandlerFunc
type DatasetHandlers struct {
	lib.DatasetMethods
	remote   *lib.RemoteMethods
	node     *p2p.QriNode
	repo     repo.Repo
	ReadOnly bool
}

// NewDatasetHandlers allocates a DatasetHandlers pointer
func NewDatasetHandlers(inst *lib.Instance, readOnly bool) *DatasetHandlers {
	dsm := lib.NewDatasetMethods(inst)
	rm := lib.NewRemoteMethods(inst)
	h := DatasetHandlers{*dsm, rm, inst.Node(), inst.Node().Repo, readOnly}
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

// ChangesHandler is a dataset single endpoint
func (h *DatasetHandlers) ChangesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost, http.MethodGet:
		if h.ReadOnly {
			readOnlyResponse(w, "/changereport")
			return
		}
		h.changesHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
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
	default:
		return ""
	}
}

func (h *DatasetHandlers) listHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.ListParams{}
	if err := UnmarshalParams(r, params); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.List(r.Context(), params)
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
	if err := util.WritePageResponse(w, res, r, params.Page()); err != nil {
		log.Infof("error list datasests response: %s", err.Error())
	}
}

func (h *DatasetHandlers) getHandler(w http.ResponseWriter, r *http.Request) {
	params := lib.GetParams{}

	err := UnmarshalParams(r, &params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	result, err := h.Get(r.Context(), &params)
	if err != nil {
		util.RespondWithError(w, err)
		return
	}
	h.replyWithGetResponse(w, r, &params, result)
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
	req := &lib.DiffParams{}
	switch r.Header.Get("Content-Type") {
	case "application/json":
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error decoding body into params: %s", err.Error()))
			return
		}
	default:
		req = &lib.DiffParams{
			LeftSide:  r.FormValue("left_path"),
			RightSide: r.FormValue("right_path"),
			Selector:  r.FormValue("selector"),
		}
	}

	res := &lib.DiffResponse{}
	if err := h.Diff(req, res); err != nil {
		fmt.Println(err)
		util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error generating diff: %s", err.Error()))
		return
	}

	util.WritePageResponse(w, res, r, util.Page{})
}

func (h *DatasetHandlers) changesHandler(w http.ResponseWriter, r *http.Request) {
	req := &lib.ChangeReportParams{}
	switch r.Header.Get("Content-Type") {
	case "application/json":
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error decoding body into params: %s", err.Error()))
			return
		}
	default:
		req = &lib.ChangeReportParams{
			LeftRefstr:  r.FormValue("left_path"),
			RightRefstr: r.FormValue("right_path"),
		}
	}

	res := &lib.ChangeReport{}
	if err := h.ChangeReport(req, res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error generating change report: %s", err.Error()))
		return
	}

	util.WritePageResponse(w, res, r, util.Page{})
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
	p := &lib.PullParams{
		Ref:     HTTPPathToQriPath(strings.TrimPrefix(r.URL.Path, "/pull/")),
		LinkDir: r.FormValue("dir"),
		Remote:  r.FormValue("remote"),
	}

	res := &dataset.Dataset{}
	err := h.Pull(p, res)
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
	ds := &dataset.Dataset{}

	if r.Header.Get("Content-Type") == "application/json" {
		err := json.NewDecoder(r.Body).Decode(ds)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		if strings.Contains(r.URL.Path, "/save/") {
			args, err := DatasetRefFromPath(r.URL.Path[len("/save/"):])
			if err != nil {
				if err == repo.ErrEmptyRef && r.FormValue("new") == "true" {
					// If saving a new dataset, name is not necessary
					err = nil
				} else {
					util.WriteErrResponse(w, http.StatusBadRequest, err)
					return
				}
			}
			if args.Peername != "" {
				ds.Peername = args.Peername
				ds.Name = args.Name
			}
		}
	} else {
		if err := formFileDataset(r, ds); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
	}

	// TODO (b5) - this should probably be handled by lib
	// DatasetMethods.Save should fold the provided dataset values *then* attempt
	// to extract a valid dataset reference from the resulting dataset,
	// and use that as a save target.
	ref := reporef.DatasetRef{
		Name:     ds.Name,
		Peername: ds.Peername,
	}

	res := &dataset.Dataset{}
	scriptOutput := &bytes.Buffer{}
	p := &lib.SaveParams{
		Ref:          ref.AliasString(),
		Dataset:      ds,
		Apply:        r.FormValue("apply") == "true",
		Private:      r.FormValue("private") == "true",
		Force:        r.FormValue("force") == "true",
		ShouldRender: !(r.FormValue("no_render") == "true"),
		NewName:      r.FormValue("new") == "true",
		BodyPath:     r.FormValue("bodypath"),
		Drop:         r.FormValue("drop"),

		ConvertFormatToPrev: true,
		ScriptOutput:        scriptOutput,
	}

	if r.FormValue("secrets") != "" {
		p.Secrets = map[string]string{}
		if err := json.Unmarshal([]byte(r.FormValue("secrets")), &p.Secrets); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("parsing secrets: %s", err))
			return
		}
	} else if ds.Transform != nil && ds.Transform.Secrets != nil {
		// TODO remove this, require API consumers to send secrets separately
		p.Secrets = ds.Transform.Secrets
	}

	if err := h.Save(p, res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
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
	ref := HTTPPathToQriPath(strings.TrimPrefix(r.URL.Path, "/remove/"))

	if remote := r.FormValue("remote"); remote != "" {
		res := &dsref.Ref{}
		err := h.remote.Remove(&lib.PushParams{
			Ref:        ref,
			RemoteName: remote,
		}, res)
		if err != nil {
			log.Error("deleting dataset from remote: %s", err.Error())
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		util.WriteResponse(w, res)
		return
	}

	p := lib.RemoveParams{
		Ref:       ref,
		Revision:  dsref.Rev{Field: "ds", Gen: -1},
		KeepFiles: r.FormValue("keep-files") == "true",
		Force:     r.FormValue("force") == "true",
	}
	if r.FormValue("all") == "true" {
		p.Revision = dsref.NewAllRevisions()
	}

	res := lib.RemoveResponse{}
	if err := h.Remove(&p, &res); err != nil {
		log.Infof("error deleting dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h DatasetHandlers) renameHandler(w http.ResponseWriter, r *http.Request) {
	p := &lib.RenameParams{}
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(p); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
	} else {
		p.Current = r.URL.Query().Get("current")
		p.Next = r.URL.Query().Get("new")
		if p.Next == "" {
			p.Next = r.URL.Query().Get("next")
		}
	}

	res := &dsref.VersionInfo{}
	if err := h.Rename(p, res); err != nil {
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
