package api

import (
	"fmt"
	"net/http"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/qri/lib"
)

// FSIHandlers connects HTTP requests to the FSI subsystem
type FSIHandlers struct {
	lib.FSIMethods
	ReadOnly bool
}

// NewFSIHandlers creates handlers that talk to qri's filesystem integration
func NewFSIHandlers(inst *lib.Instance, readOnly bool) FSIHandlers {
	return FSIHandlers{
		FSIMethods: *lib.NewFSIMethods(inst),
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
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		if h.ReadOnly {
			readOnlyResponse(w, "/fsi")
			return
		}
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
