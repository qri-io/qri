package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
)

// FSIHandlers connects HTTP requests to the FSI subsystem
type FSIHandlers struct {
	lib.FSIMethods
	dsm      *lib.DatasetRequests
	ReadOnly bool
}

// NewFSIHandlers creates handlers that talk to qri's filesystem integration
func NewFSIHandlers(inst *lib.Instance, readOnly bool) FSIHandlers {
	return FSIHandlers{
		FSIMethods: *lib.NewFSIMethods(inst),
		dsm:        lib.NewDatasetRequests(inst.Node(), nil),
		ReadOnly:   readOnly,
	}
}

// LinksHandler is the endpoint for getting the list of fsi-linked datasets
func (h *FSIHandlers) LinksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		if h.ReadOnly {
			readOnlyResponse(w, "/fsilinks")
			return
		}
		h.linksHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *FSIHandlers) linksHandler(w http.ResponseWriter, r *http.Request) {
	p := false
	res := []*lib.FSILink{}
	if err := h.Links(&p, &res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error listing links: %s", err.Error()))
		return
	}
	util.WriteResponse(w, res)
}

// StatusHandler is the endpoint for getting the status of a linked dataset
func (h *FSIHandlers) StatusHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		if h.ReadOnly {
			readOnlyResponse(w, "/dsstatus")
			return
		}
		h.statusHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *FSIHandlers) statusHandler(w http.ResponseWriter, r *http.Request) {
	ref, err := DatasetRefFromPath(r.URL.Path[len("/dsstatus"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("bad reference: %s", err.Error()))
		return
	}

	alias := ref.AliasString()
	res := []lib.StatusItem{}
	if err = h.AliasStatus(&alias, &res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error getting status: %s", err.Error()))
		return
	}
	util.WriteResponse(w, res)
}

// DatasetHandler returns an fsi-linked dataset
func (h *FSIHandlers) DatasetHandler(w http.ResponseWriter, r *http.Request) {
	if h.ReadOnly {
		readOnlyResponse(w, "/fsi")
		return
	}

	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.datasetHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *FSIHandlers) datasetHandler(w http.ResponseWriter, r *http.Request) {
	ref, err := DatasetRefFromPath(r.URL.Path[len("/fsi"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("bad reference: %s", err.Error()))
		return
	}

	str := ref.String()
	if err := h.FSIDatasetForRef(&str, &ref); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, ref)
}

// BodyHandler reads an fsi-linked dataset body
func (h *FSIHandlers) BodyHandler(w http.ResponseWriter, r *http.Request) {
	if h.ReadOnly {
		readOnlyResponse(w, "/fsi/body")
		return
	}

	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.bodyHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *FSIHandlers) bodyHandler(w http.ResponseWriter, r *http.Request) {
	ref, err := DatasetRefFromPath(r.URL.Path[len("/fsi/body"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("bad reference: %s", err.Error()))
		return
	}
	listParams := lib.ListParamsFromRequest(r)

	p := &lib.FSIBodyParams{
		Path:   ref.String(),
		Format: "json",
		Limit:  listParams.Limit,
		Offset: listParams.Offset,
		All:    r.FormValue("all") == "true",
	}
	res := []byte{}
	if err := h.FSIDatasetBody(p, &res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, json.RawMessage(res))
}

// SaveHandler saves datasets from the filesystem via an API call
func (h *FSIHandlers) SaveHandler(w http.ResponseWriter, r *http.Request) {
	if h.ReadOnly {
		readOnlyResponse(w, "/fsi/save")
		return
	}

	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST":
		h.saveHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *FSIHandlers) saveHandler(w http.ResponseWriter, r *http.Request) {
	ref, err := DatasetRefFromPath(r.URL.Path[len("/fsi/save"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("bad reference: %s", err.Error()))
		return
	}

	fds := &dataset.Dataset{}
	if err := json.NewDecoder(r.Body).Decode(fds); err != nil && err.Error() != "EOF" {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	str := ref.String()
	if err := h.FSIDatasetForRef(&str, &ref); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	ds := ref.Dataset
	// ds.Assign(fds)

	path := ds.Path
	bodyPath := ds.BodyPath

	if ds.Transform != nil && ds.Transform.ScriptPath != "" {
		ds.Transform.ScriptPath = filepath.Join(path, ds.Transform.ScriptPath)
		fmt.Println(ds.Transform)
	}
	if ds.Viz != nil && ds.Viz.ScriptPath != "" {
		ds.Viz.ScriptPath = filepath.Join(path, ds.Viz.ScriptPath)
	}

	ds.Path = ""
	ds.PreviousPath = ""
	ds.BodyPath = ""

	fmt.Println(path, ds.BodyPath)
	fmt.Printf("%#v\n", ds)

	p := &lib.SaveParams{
		Ref:      ref.AliasString(),
		Dataset:  ds,
		BodyPath: bodyPath,
		// TODO (b5) - save has *many* other params that need to be filled
		// in here for feature parity
	}
	res := &repo.DatasetRef{}
	if err := h.dsm.Save(p, res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}
