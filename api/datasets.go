package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/schema"
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

// StatsHandler gets stats about the dataset
func (h *DatasetHandlers) StatsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.statsHandler(w, r)
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

// snoop reads from an io.ReadCloser and restores it so it can be read again
func snoop(body *io.ReadCloser) (io.ReadCloser, error) {
	if body != nil && *body != nil {
		result, err := ioutil.ReadAll(*body)
		(*body).Close()

		if err != nil {
			return nil, err
		}
		if len(result) == 0 {
			return nil, io.EOF
		}

		*body = ioutil.NopCloser(bytes.NewReader(result))
		return ioutil.NopCloser(bytes.NewReader(result)), nil
	}
	return nil, io.EOF
}

var decoder = schema.NewDecoder()

// UnmarshalParams deserialzes a lib req params stuct pointer from an HTTP
// request
func UnmarshalParams(r *http.Request, p interface{}) error {
	// TODO(arqu): this should be set on the global decoder
	// probably means the decoder should live in the API struct or somewhere similar
	decoder.IgnoreUnknownKeys(true)
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		defer func() {
			if defSetter, ok := p.(lib.NZDefaultSetter); ok {
				defSetter.SetNonZeroDefaults()
			}
		}()

		if r.Header.Get("Content-Type") == jsonContentType {
			body, err := snoop(&r.Body)
			if err != nil && err != io.EOF {
				return err
			}
			// this avoids resolving on empty body requests
			// and tries to handle it almost like a GET
			if err != io.EOF {
				return json.NewDecoder(body).Decode(p)
			}
		}
	}

	if ru, ok := p.(lib.RequestUnmarshaller); ok {
		return ru.UnmarshalFromRequest(r)
	}

	if err := r.ParseForm(); err != nil {
		return err
	}
	return decoder.Decode(p, r.Form)
}

func (h *DatasetHandlers) listHandler(w http.ResponseWriter, r *http.Request) {
	args := &lib.ListParams{}
	if err := UnmarshalParams(r, args); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.List(r.Context(), args)
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
	if err := util.WritePageResponse(w, res, r, args.Page()); err != nil {
		log.Infof("error list datasests response: %s", err.Error())
	}
}

func (h *DatasetHandlers) getHandler(w http.ResponseWriter, r *http.Request) {
	params := lib.GetParams{}
	args := &GetReqArgs{}

	err := UnmarshalParams(r, &params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	if !params.Proxy {
		args, err = parseGetReqArgs(r, strings.TrimPrefix(r.URL.Path, "/get/"))
		if err != nil {
			util.RespondWithError(w, err)
			return
		}
		params = args.Params
	} else {
		// TODO(arqu): get rid of this once we move the proxy call above lib
		args.Params = params
		ref, err := dsref.Parse(params.Refstr)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
		}
		args.Ref = ref
		// TODO(arqu): fix raw download on proxy. Requires GetReqArgs to be proxied too
		args.RawDownload = false
	}

	result, err := h.Get(r.Context(), &params)
	if err != nil {
		util.RespondWithError(w, err)
		return
	}

	h.replyWithGetResponse(w, r, &params, result, args)
}

// replyWithGetResponse writes an http response back to the client, based upon what sort of
// response they requested. Handles raw file downloads (without response wrappers), zip downloads,
// body pagination, as well as normal head responses. Input logic has already been handled
// before this function, so errors should not commonly happen.
func (h *DatasetHandlers) replyWithGetResponse(w http.ResponseWriter, r *http.Request, params *lib.GetParams, result *lib.GetResult, args *GetReqArgs) {

	// Convert components with scriptPaths (transform, readme, viz) in scriptBytes
	if err := inlineScriptsToBytes(result.Dataset); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	resultFormat := params.Format
	if resultFormat == "" {
		resultFormat = result.Dataset.Structure.Format
	}

	// Format zip returns zip file without a json wrapper
	if resultFormat == "zip" {
		zipFilename := fmt.Sprintf("%s.zip", args.Ref.Name)
		w.Header().Set("Content-Type", extensionToMimeType(".zip"))
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", zipFilename))
		w.Write(result.Bytes)
		return
	}

	// RawDownload is true if download=true or the "Accept: text/csv" header is set
	if args.RawDownload {
		filename, err := archive.GenerateFilename(result.Dataset, resultFormat)
		if err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", extensionToMimeType("."+resultFormat))
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		w.Write(result.Bytes)
		return
	}

	if params.Selector != "" {
		page := util.PageFromRequest(r)
		dataResponse := DataResponse{
			Path: result.Dataset.BodyPath,
			Data: json.RawMessage(result.Bytes),
		}
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
	params := lib.SaveParams{}
	err := UnmarshalParams(r, &params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	if params.Proxy {
		res, err := h.Save(r.Context(), &params)
		if err != nil {
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

		util.WriteMessageResponse(w, "", resRef)
		return
	}

	if params.Dataset != nil || r.Header.Get("Content-Type") == "application/json" {
		if params.Dataset == nil {
			params.Dataset = &dataset.Dataset{}
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
				params.Dataset.Peername = args.Peername
				params.Dataset.Name = args.Name
			}
		}
	} else {
		if params.Dataset == nil {
			params.Dataset = &dataset.Dataset{}
		}
		if err := formFileDataset(r, params.Dataset); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
	}

	// TODO (b5) - this should probably be handled by lib
	// DatasetMethods.Save should fold the provided dataset values *then* attempt
	// to extract a valid dataset reference from the resulting dataset,
	// and use that as a save target.
	ref := reporef.DatasetRef{
		Name:     params.Dataset.Name,
		Peername: params.Dataset.Peername,
	}

	scriptOutput := &bytes.Buffer{}
	p := &lib.SaveParams{
		Ref:          ref.AliasString(),
		Dataset:      params.Dataset,
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
	} else if params.Dataset.Transform != nil && params.Dataset.Transform.Secrets != nil {
		// TODO remove this, require API consumers to send secrets separately
		p.Secrets = params.Dataset.Transform.Secrets
	}

	res, err := h.Save(r.Context(), p)
	if err != nil {
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
	params := lib.RemoveParams{}
	err := UnmarshalParams(r, &params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	if params.Proxy {
		res, err := h.Remove(r.Context(), &params)
		if err != nil {
			log.Infof("error deleting dataset: %s", err.Error())
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}

		util.WriteResponse(w, res)
		return
	}

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

	res, err := h.Remove(r.Context(), &p)
	if err != nil {
		log.Infof("error deleting dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h DatasetHandlers) renameHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.RenameParams{}
	err := UnmarshalParams(r, params)
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

// DataResponse is the struct used to respond to api requests made to the /body endpoint
// It is necessary because we need to include the 'path' field in the response
type DataResponse struct {
	Path string          `json:"path"`
	Data json.RawMessage `json:"data"`
}

// GetReqArgs is the result of parsing parameters and other control options from the http request
type GetReqArgs struct {
	Params      lib.GetParams
	Ref         dsref.Ref
	RawDownload bool
}

// parseGetReqArgs creates getParams from a request
func parseGetReqArgs(r *http.Request, reqPath string) (*GetReqArgs, error) {
	hasBodyCsvSuffix := false
	if strings.HasSuffix(reqPath, "/body.csv") {
		reqPath = strings.TrimSuffix(reqPath, "/body.csv")
		hasBodyCsvSuffix = true
	}

	refStr := HTTPPathToQriPath(reqPath)
	ref, err := dsref.Parse(refStr)
	if err != nil {
		return nil, err
	}

	if ref.Username == "me" {
		return nil, util.NewAPIError(http.StatusBadRequest, "username \"me\" not allowed")
	}

	// page and pageSize
	listParams := lib.ListParamsFromRequest(r)

	rawDownload := r.FormValue("download") == "true"
	format := r.FormValue("format")
	component := r.FormValue("component")

	getAll := r.FormValue("all") == "true"
	offset := listParams.Offset
	limit := listParams.Limit
	if offset == 0 && limit == -1 {
		getAll = true
	}

	// This HTTP header sets the format to csv, and removes the json wrapper
	if arrayContains(r.Header["Accept"], "text/csv") {
		if format != "" && format != "csv" {
			return nil, util.NewAPIError(http.StatusBadRequest, fmt.Sprintf("format %q conflicts with header \"Accept: text/csv\"", format))
		}
		format = "csv"
		rawDownload = true
	}

	// The body.csv suffix is a convenience feature to get the entire body as a csv
	if hasBodyCsvSuffix {
		format = "csv"
		rawDownload = true
		getAll = true
	}

	// API is a json api, so the default format is json
	if format == "" {
		format = "json"
	}

	// Raw download must mean the body
	if rawDownload {
		if component != "" && component != "body" {
			return nil, util.NewAPIError(http.StatusBadRequest, "cannot download component aside from \"body\"")
		}
		component = "body"
	}

	// Setting any other format, without it being a raw download, is an error
	if !rawDownload {
		if format != "json" && format != "zip" {
			return nil, util.NewAPIError(http.StatusBadRequest, "only supported formats are \"json\" and \"zip\", unless using download parameter or Accept header is set to \"text/csv\"")
		}
	}

	params := lib.GetParams{
		Refstr:   ref.String(),
		Format:   format,
		Selector: component,
		Limit:    listParams.Limit,
		Offset:   listParams.Offset,
		All:      getAll,
		Remote:   r.FormValue("remote"),
	}
	args := GetReqArgs{
		Ref:         ref,
		RawDownload: rawDownload,
		Params:      params,
	}

	return &args, nil
}

func (h DatasetHandlers) statsHandler(w http.ResponseWriter, r *http.Request) {
	p := lib.GetParams{
		Refstr:   HTTPPathToQriPath(strings.TrimPrefix(r.URL.Path, "/stats/")),
		Selector: "stats",
	}
	res, err := h.Get(r.Context(), &p)
	if err != nil {
		if err == repo.ErrNoHistory {
			util.WriteErrResponse(w, http.StatusUnprocessableEntity, err)
			return
		}
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	statsMap := &[]map[string]interface{}{}
	if err := json.Unmarshal(res.Bytes, statsMap); err != nil {
		log.Errorf("error unmarshalling stats: %s", err)
		util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error writing stats"))
		return
	}
	if err := util.WriteResponse(w, statsMap); err != nil {
		log.Infof("error writing response: %s", err.Error())
	}
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

func arrayContains(subject []string, target string) bool {
	for _, v := range subject {
		if v == target {
			return true
		}
	}
	return false
}
