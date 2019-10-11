package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/dsref"
)

// DatasetHandlers wraps a requests struct to interface with http.HandlerFunc
type DatasetHandlers struct {
	lib.DatasetRequests
	node     *p2p.QriNode
	repo     repo.Repo
	ReadOnly bool
}

// NewDatasetHandlers allocates a DatasetHandlers pointer
func NewDatasetHandlers(inst *lib.Instance, readOnly bool) *DatasetHandlers {
	req := lib.NewDatasetRequestsInstance(inst)
	h := DatasetHandlers{*req, inst.Node(), inst.Node().Repo, readOnly}
	return &h
}

// ListHandler is a dataset list endpoint
func (h *DatasetHandlers) ListHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
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
		if h.ReadOnly {
			readOnlyResponse(w, "/me/")
			return
		}
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
		if h.ReadOnly {
			readOnlyResponse(w, "/diff")
			return
		}
		h.diffHandler(w, r)
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

// BodyHandler gets the contents of a dataset
func (h *DatasetHandlers) BodyHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		if h.ReadOnly {
			readOnlyResponse(w, "/body/")
			return
		}
		h.bodyHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// UnpackHandler unpacks a zip file and sends it back as json
func (h *DatasetHandlers) UnpackHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST":
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

// ZipDatasetHandler is the endpoint for getting a zip archive of a dataset
func (h *DatasetHandlers) ZipDatasetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		if h.ReadOnly {
			readOnlyResponse(w, "/export/")
			return
		}
		h.zipDatasetHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *DatasetHandlers) zipDatasetHandler(w http.ResponseWriter, r *http.Request) {
	ref := HTTPPathToQriPath(r.URL.Path[len("/export"):])
	// default is zipped
	zipped := r.FormValue("zipped") != "false"
	format := r.FormValue("format")
	tmpDir, err := ioutil.TempDir(os.TempDir(), "api_export")
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	params := lib.ExportParams{Ref: ref, TargetDir: tmpDir, Format: format, Zipped: zipped}

	var fileWritten string
	req := lib.NewExportRequests(h.node, nil)
	err = req.Export(&params, &fileWritten)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	f, err := os.Open(filepath.Join(tmpDir, fileWritten))
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", extensionToMimeType(path.Ext(fileWritten)))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", path.Base(fileWritten)))
	w.Write(bytes)
}

func extensionToMimeType(ext string) string {
	switch ext {
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
	args := lib.ListParamsFromRequest(r)
	args.OrderBy = "created"

	args.Term = r.FormValue("term")

	res := []repo.DatasetRef{}
	if err := h.List(&args, &res); err != nil {
		log.Infof("error listing datasets: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := util.WritePageResponse(w, res, r, args.Page()); err != nil {
		log.Infof("error list datasests response: %s", err.Error())
	}
}

// TODO (ramfox): we have two places where `get` is happening, here and at root.go
// we should deprecate the `/me` endpoint (and this handler)
// and have the root check to see if `me` is the peername
// if we are in read-only mode, we should error,
// otherwise, resolve the peername and proceed as normal
func (h *DatasetHandlers) getHandler(w http.ResponseWriter, r *http.Request) {
	p := lib.GetParams{
		Path:   HTTPPathToQriPath(r.URL.Path),
		UseFSI: r.FormValue("fsi") == "true",
	}
	res := lib.GetResult{}
	err := h.Get(&p, &res)
	if err != nil {
		if err == repo.ErrNoHistory || err == fsi.ErrNoLink {
			util.WriteErrResponse(w, http.StatusUnprocessableEntity, err)
			return
		}
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	// TODO (b5) - remove this. res.Ref should be used instead
	ref := repo.DatasetRef{
		Peername:  res.Dataset.Peername,
		ProfileID: profile.ID(res.Dataset.ProfileID),
		Name:      res.Dataset.Name,
		Path:      res.Dataset.Path,
		FSIPath:   res.Ref.FSIPath,
		Published: res.Ref.Published,
		Dataset:   res.Dataset,
	}
	util.WriteResponse(w, ref)
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
			LeftPath:  r.FormValue("left_path"),
			RightPath: r.FormValue("right_path"),
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

	res := []repo.DatasetRef{}
	if err := h.List(&p, &res); err != nil {
		log.Infof("error listing peer's datasets: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := util.WritePageResponse(w, res, r, p.Page()); err != nil {
		log.Infof("error list datasests response: %s", err.Error())
	}
}

func (h *DatasetHandlers) addHandler(w http.ResponseWriter, r *http.Request) {
	ref, err := DatasetRefFromPath(r.URL.Path[len("/add"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	// TODO (b5) - move this into lib.Add
	if ref.Peername == "" || ref.Name == "" {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("need peername and dataset name: '/add/[peername]/[datasetname]'"))
		return
	}

	p := &lib.AddParams{
		Ref:     ref.String(),
		LinkDir: r.FormValue("dir"),
	}

	res := repo.DatasetRef{}
	err = h.Add(p, &res)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
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
				util.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			if args.Peername != "" {
				ds.Peername = args.Peername
				ds.Name = args.Name
			}
		}
	} else {
		if err := dsutil.FormFileDataset(r, ds); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
	}

	// TODO (b5) - this should probably be handled by lib
	// DatasetMethods.Save should fold the provided dataset values *then* attempt
	// to extract a valid dataset reference from the resulting dataset,
	// and use that as a save target.
	ref := repo.DatasetRef{
		Name:     ds.Name,
		Peername: ds.Peername,
	}

	res := &repo.DatasetRef{}
	scriptOutput := &bytes.Buffer{}
	p := &lib.SaveParams{
		Ref:          ref.AliasString(),
		Dataset:      ds,
		Private:      r.FormValue("private") == "true",
		DryRun:       r.FormValue("dry_run") == "true",
		ReturnBody:   r.FormValue("return_body") == "true",
		Force:        r.FormValue("force") == "true",
		ShouldRender: !(r.FormValue("no_render") == "true"),
		ReadFSI:      r.FormValue("fsi") == "true",
		WriteFSI:     r.FormValue("fsi") == "true",

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
	res.Dataset.BodyPath = filepath.Base(res.Dataset.BodyPath)

	msg := scriptOutput.String()
	util.WriteMessageResponse(w, msg, res)
}

func (h *DatasetHandlers) removeHandler(w http.ResponseWriter, r *http.Request) {
	p := lib.RemoveParams{
		Ref:            HTTPPathToQriPath(r.URL.Path[len("/remove"):]),
		Revision:       dsref.Rev{Field: "ds", Gen: -1},
		Unlink:         r.FormValue("unlink") == "true",
		DeleteFSIFiles: r.FormValue("files") == "true",
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

// RenameReqParams is an encoding struct
// its intent is to be a more user-friendly structure for the api endpoint
// that will map to and from the lib.RenameParams struct
type RenameReqParams struct {
	Current string
	New     string
}

func (h DatasetHandlers) renameHandler(w http.ResponseWriter, r *http.Request) {
	reqParams := &RenameReqParams{}
	p := &lib.RenameParams{}
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
	p = &lib.RenameParams{
		Current: current,
		New:     n,
	}

	res := &repo.DatasetRef{}
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

// DataResponse is the struct used to respond to api requests made to the /data endpoint
// It is necessary because we need to include the 'path' field in the response
type DataResponse struct {
	Path string          `json:"path"`
	Data json.RawMessage `json:"data"`
}

// getParamsFromRequest creates getParams from a request. It's currently only used for paginating dataset bodies
func getParamsFromRequest(r *http.Request, readOnly bool, path string) (*lib.GetParams, error) {
	listParams := lib.ListParamsFromRequest(r)
	download := r.FormValue("download") == "true"
	format := "json"
	if download {
		format = r.FormValue("format")
	}
	// if download is not set, and format is set, make sure the user knows that
	// setting format won't do anything
	if !download && r.FormValue("format") != "" && r.FormValue("format") != "json" {
		return nil, fmt.Errorf("the format must be json if used without the download parameter")
	}

	p := &lib.GetParams{
		Path:     path,
		Format:   format,
		Selector: "body",
		UseFSI:   r.FormValue("fsi") == "true",
		Limit:    listParams.Limit,
		Offset:   listParams.Offset,
		All:      r.FormValue("all") == "true" && !readOnly,
	}

	if !readOnly {
		offset, offsetErr := util.ReqParamInt("offset", r)
		limit, limitErr := util.ReqParamInt("limit", r)

		if offsetErr == nil || limitErr == nil {
			if limitErr != nil {
				limit = util.DefaultPageSize
			}
			if offsetErr != nil {
				offset = 0
			}
			p.Limit = limit
			p.Offset = offset
			if limit == -1 && offset == 0 {
				p.All = true
			}
		}
		// if we request all explicitly, or if offset is zero and limit is -1
		// return all rows
		p.All = r.FormValue("all") == "true" || (p.Offset == 0 && p.Limit == -1)
	}
	return p, nil
}

func (h DatasetHandlers) bodyHandler(w http.ResponseWriter, r *http.Request) {
	refStr := HTTPPathToQriPath(r.URL.Path[len("/body/"):])
	p, err := getParamsFromRequest(r, h.ReadOnly, refStr)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	result := &lib.GetResult{}
	if err := h.Get(p, result); err != nil {
		if err == repo.ErrNoHistory {
			util.WriteErrResponse(w, http.StatusUnprocessableEntity, err)
			return
		}
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	download := r.FormValue("download") == "true"
	if download {
		filename, err := lib.GenerateFilename(result.Dataset, p.Format)
		if err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", extensionToMimeType("."+p.Format))
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		w.Write(result.Bytes)
		return
	}

	page := util.PageFromRequest(r)
	path := result.Dataset.BodyPath
	if p.UseFSI {
		path = result.Dataset.Path
	}

	dataResponse := DataResponse{
		Path: path,
		Data: json.RawMessage(result.Bytes),
	}
	if err := util.WritePageResponse(w, dataResponse, r, page); err != nil {
		log.Infof("error writing response: %s", err.Error())
	}
}

func (h DatasetHandlers) unpackHandler(w http.ResponseWriter, r *http.Request, postData []byte) {
	contents, err := dsutil.UnzipGetContents(postData)
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
