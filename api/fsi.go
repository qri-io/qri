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

	res := []lib.StatusItem{}
	if ref.Path != "" {
		refStr := ref.String()
		if err = h.StoredStatus(&refStr, &res); err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error getting status: %s", err.Error()))
			return
		}
	} else {
		alias := ref.AliasString()
		if err = h.AliasStatus(&alias, &res); err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error getting status: %s", err.Error()))
			return
		}
	}

	util.WriteResponse(w, res)
}

// InitHandler creates a new FSI-linked dataset
func (h *FSIHandlers) InitHandler(w http.ResponseWriter, r *http.Request) {
	if h.ReadOnly {
		readOnlyResponse(w, "/init")
		return
	}

	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST":
		h.initHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *FSIHandlers) initHandler(w http.ResponseWriter, r *http.Request) {
	p := &lib.InitFSIDatasetParams{
		Filepath: r.FormValue("filepath"),
		Name:     r.FormValue("name"),
		Format:   r.FormValue("format"),
	}

	var name string
	if err := h.InitDataset(p, &name); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	util.WriteResponse(w, map[string]string{"ref": name})
}

// CheckoutHandler invokes checkout via an API call
func (h *FSIHandlers) CheckoutHandler(w http.ResponseWriter, r *http.Request) {
	if h.ReadOnly {
		readOnlyResponse(w, "/checkout")
		return
	}

	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST":
		h.checkoutHandler(w, r)
	default:
		util.NotFoundHandler(w,r)
	}
}

func (h *FSIHandlers) checkoutHandler(w http.ResponseWriter, r *http.Request) {
	p := &lib.CheckoutParams{
		Dir: r.FormValue("dir"),
		Ref: r.FormValue("ref"),
	}

	var res string
	if err := h.Checkout(p, &res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, p)
}
