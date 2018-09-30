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

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/dsdiff"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// DatasetHandlers wraps a requests struct to interface with http.HandlerFunc
type DatasetHandlers struct {
	lib.DatasetRequests
	node     *p2p.QriNode
	repo     repo.Repo
	ReadOnly bool
}

// NewDatasetHandlers allocates a DatasetHandlers pointer
func NewDatasetHandlers(node *p2p.QriNode, readOnly bool) *DatasetHandlers {
	req := lib.NewDatasetRequests(node, nil)
	h := DatasetHandlers{*req, node, node.Repo, readOnly}
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
	args, err := DatasetRefFromPath(r.URL.Path[len("/export"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	res := &repo.DatasetRef{}
	err = h.Get(&args, res)
	if err != nil {
		log.Infof("error getting dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	ds, err := res.DecodeDataset()
	if err != nil {
		log.Infof("error decoding dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("filename=\"%s.zip\"", "dataset"))
	dsutil.WriteZipArchive(h.repo.Store(), ds, w)
}

func (h *DatasetHandlers) listHandler(w http.ResponseWriter, r *http.Request) {
	args := lib.ListParamsFromRequest(r)
	args.OrderBy = "created"

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

	left, err := DatasetRefFromPath(d.Left)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error getting datasetRef from left path: %s", err.Error()))
		return
	}

	right, err := DatasetRefFromPath(d.Right)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error getting datasetRef from right path: %s", err.Error()))
		return
	}

	diffs := make(map[string]*dsdiff.SubDiff)
	p := &lib.DiffParams{
		Left:    left,
		Right:   right,
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

func (h *DatasetHandlers) initHandler(w http.ResponseWriter, r *http.Request) {
	dsp := &dataset.DatasetPod{}
	switch r.Header.Get("Content-Type") {
	case "application/json":
		if err := json.NewDecoder(r.Body).Decode(dsp); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("error decoding body into params: %s", err.Error()))
			return
		}

		if dsp.BodyPath == "" {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("if adding dataset using json, body of request must have 'bodyPath' field"))
			return
		}

	default:
		cleanup, err := formFileDataset(dsp, r)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		defer cleanup()
	}

	// TODO - fix this awful mess, ioes needs some method for piping it's output
	prev := h.node.LocalStreams
	defer func() {
		h.node.LocalStreams = prev
	}()

	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	s := ioes.NewIOStreams(in, out, errOut)
	h.node.LocalStreams = s

	res := &repo.DatasetRef{}
	p := &lib.SaveParams{
		Dataset:    dsp,
		Private:    r.FormValue("private") == "true",
		DryRun:     r.FormValue("dry_run") == "true",
		ReturnBody: r.FormValue("return_body") == "true",
	}
	if err := h.New(p, res); err != nil {
		log.Infof("error initializing dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	if p.ReturnBody {
		if err := addBodyFile(res); err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
	}

	// util.WriteResponse(w, res)
	env := map[string]interface{}{
		"meta": map[string]interface{}{
			"code":    http.StatusOK,
			"message": string(out.Bytes()),
		},
		"data": res.Dataset,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(env)
}

func formFileDataset(dsp *dataset.DatasetPod, r *http.Request) (cleanup func(), err error) {
	var rmFiles []*os.File
	cleanup = func() {
		for _, f := range rmFiles {
			// TODO - log error?
			os.Remove(f.Name())
		}
	}

	datafile, dataHeader, err := r.FormFile("file")
	if err != nil && err != http.ErrMissingFile {
		err = fmt.Errorf("error opening dataset file: %s", err)
		return
	}
	if datafile != nil {
		switch strings.ToLower(filepath.Ext(dataHeader.Filename)) {
		case ".yaml", ".yml":
			var data []byte
			data, err = ioutil.ReadAll(datafile)
			if err != nil {
				err = fmt.Errorf("error reading dataset file: %s", err)
				return
			}
			if err = dsutil.UnmarshalYAMLDatasetPod(data, dsp); err != nil {
				err = fmt.Errorf("error unmarshaling yaml file: %s", err)
				return
			}
		case ".json":
			if err = json.NewDecoder(datafile).Decode(dsp); err != nil {
				err = fmt.Errorf("error decoding json file: %s", err)
				return
			}
		}
	}

	tfFile, _, err := r.FormFile("transform")
	if err != nil && err != http.ErrMissingFile {
		err = fmt.Errorf("error opening transform file: %s", err)
		return
	}
	if tfFile != nil {
		// TODO - this assumes a skylark / starlark transform file
		f, e := ioutil.TempFile("", "transform")
		if e != nil {
			err = e
			return
		}
		rmFiles = append(rmFiles, f)
		io.Copy(f, tfFile)
		if dsp.Transform == nil {
			dsp.Transform = &dataset.TransformPod{}
		}
		dsp.Transform.Syntax = "skylark"
		dsp.Transform.ScriptPath = f.Name()
	}

	vizFile, _, err := r.FormFile("viz")
	if err != nil && err != http.ErrMissingFile {
		err = fmt.Errorf("error opening viz file: %s", err)
		return
	}
	if vizFile != nil {
		// TODO - this assumes an html viz file
		f, e := ioutil.TempFile("", "viz")
		if e != nil {
			err = e
			return
		}
		rmFiles = append(rmFiles, f)
		io.Copy(f, vizFile)
		if dsp.Viz == nil {
			dsp.Viz = &dataset.Viz{}
		}
		dsp.Viz.Format = "html"
		dsp.Viz.ScriptPath = f.Name()
	}

	dsp.Peername = r.FormValue("peername")
	dsp.Name = r.FormValue("name")
	dsp.BodyPath = r.FormValue("body_path")

	bodyfile, bodyHeader, err := r.FormFile("body")
	if err != nil && err != http.ErrMissingFile {
		err = fmt.Errorf("error opening body file: %s", err)
		return
	}
	if bodyfile != nil {
		path := filepath.Join(os.TempDir(), bodyHeader.Filename)
		f, e := os.Create(path)
		if e != nil {
			err = fmt.Errorf("error writing body file: %s", e.Error())
			return
		}
		rmFiles = append(rmFiles, f)
		io.Copy(f, bodyfile)
		f.Close()
		dsp.BodyPath = path
	}

	return
}

func addBodyFile(res *repo.DatasetRef) error {
	if res.Dataset.Structure.Format == dataset.JSONDataFormat.String() {
		// TODO - this'll only work for JSON responses
		data, err := ioutil.ReadAll(res.Dataset.Body.(io.Reader))
		if err != nil {
			return err
		}
		res.Dataset.Body = json.RawMessage(data)
	} else {
		if file, ok := res.Dataset.Body.(cafs.File); ok {
			in := &dataset.Structure{}
			if err := in.Decode(res.Dataset.Structure); err != nil {
				return err
			}

			st := &dataset.Structure{}
			st.Assign(in, &dataset.Structure{
				Format: dataset.JSONDataFormat,
				Schema: in.Schema,
			})

			data, err := actions.ConvertBodyFile(file, in, st, 0, 0, true)
			if err != nil {
				return fmt.Errorf("converting body file to JSON: %s", err)
			}
			res.Dataset.Body = json.RawMessage(data)
		} else {
			log.Error("response body isn't a cafs.File")
		}
	}
	return nil
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

func (h *DatasetHandlers) saveHandler(w http.ResponseWriter, r *http.Request) {
	dsp := &dataset.DatasetPod{}

	if r.Header.Get("Content-Type") == "application/json" {
		err := json.NewDecoder(r.Body).Decode(dsp)
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
				dsp.Peername = args.Peername
				dsp.Name = args.Name
			}
		}
	} else {
		cleanup, err := formFileDataset(dsp, r)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		defer cleanup()
	}

	res := &repo.DatasetRef{}
	p := &lib.SaveParams{
		Dataset: dsp,
		Private: r.FormValue("private") == "true",
	}
	if err := h.Save(p, res); err != nil {
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
		log.Infof("error deleting dataset: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, ref)
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

// default number of entries to limit to when reading
// TODO - should move this into lib
const defaultDataLimit = 100

// DataResponse is the struct used to respond to api requests made to the /data endpoint
// It is necessary because we need to include the 'path' field in the response
type DataResponse struct {
	Path string          `json:"path"`
	Data json.RawMessage `json:"data"`
}

func (h DatasetHandlers) bodyHandler(w http.ResponseWriter, r *http.Request) {
	d, err := DatasetRefFromPath(r.URL.Path[len("/body"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	err = repo.CanonicalizeDatasetRef(h.repo, &d)
	if err != nil && err != repo.ErrNotFound {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
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

	p := &lib.LookupParams{
		Path:   d.Path,
		Format: dataset.JSONDataFormat,
		Limit:  limit,
		Offset: offset,
		All:    r.FormValue("all") == "true" && limit == defaultDataLimit && offset == 0,
	}

	result := &lib.LookupResult{}
	if err := h.LookupBody(p, result); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	page := util.PageFromRequest(r)
	dataResponse := DataResponse{
		Path: result.Path,
		Data: json.RawMessage(result.Data),
	}
	if err := util.WritePageResponse(w, dataResponse, r, page); err != nil {
		log.Infof("error writing response: %s", err.Error())
	}
}
