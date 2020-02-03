package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
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
func (h *FSIHandlers) StatusHandler(routePrefix string) http.HandlerFunc {
	handleStatus := h.statusHandler(routePrefix)

	return func(w http.ResponseWriter, r *http.Request) {
		if h.ReadOnly {
			readOnlyResponse(w, routePrefix)
			return
		}

		switch r.Method {
		case "OPTIONS":
			util.EmptyOkHandler(w, r)
		default:
			util.NotFoundHandler(w, r)
		case "GET":
			handleStatus(w, r)
		}
	}
}

func (h *FSIHandlers) statusHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		useFSI := r.FormValue("fsi") == "true"
		ref, err := DatasetRefFromPath(r.URL.Path[len(routePrefix):])
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("bad reference: %s", err.Error()))
			return
		}

		res := []lib.StatusItem{}
		if useFSI {
			alias := ref.AliasString()
			err := h.StatusForAlias(&alias, &res)
			// Won't return ErrNoHistory.
			if err != nil {
				util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error getting status: %s", err.Error()))
				return
			}
			util.WriteResponse(w, res)
			return
		}

		refStr := ref.String()
		err = h.StatusAtVersion(&refStr, &res)
		if err != nil {
			if err == repo.ErrNoHistory {
				util.WriteErrResponse(w, http.StatusUnprocessableEntity, err)
				return
			}
			util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error getting status: %s", err.Error()))
			return
		}
		util.WriteResponse(w, res)
	}
}

// InitHandler creates a new FSI-linked dataset
func (h *FSIHandlers) InitHandler(routePrefix string) http.HandlerFunc {
	handleInit := h.initHandler(routePrefix)

	return func(w http.ResponseWriter, r *http.Request) {
		if h.ReadOnly {
			readOnlyResponse(w, "/init")
			return
		}

		switch r.Method {
		case "OPTIONS":
			util.EmptyOkHandler(w, r)
		case "POST":
			handleInit(w, r)
		default:
			util.NotFoundHandler(w, r)
		}
	}
}

func (h *FSIHandlers) initHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Backwards compatibility shim for now, can use either "dir" or "filepath".
		// TODO: Update desktop to always use "dir", delete "filepath".
		dir := r.FormValue("dir")
		if dir == "" && r.FormValue("filepath") != "" {
			dir = r.FormValue("filepath")
		}
		p := &lib.InitFSIDatasetParams{
			Dir:            dir,
			Name:           r.FormValue("name"),
			Format:         r.FormValue("format"),
			Mkdir:          r.FormValue("mkdir"),
			SourceBodyPath: r.FormValue("sourcebodypath"),
		}

		var name string
		if err := h.InitDataset(p, &name); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		// Get code taken
		// taken from ./root.go
		gp := lib.GetParams{
			Path:   name,
			UseFSI: true,
		}
		res := lib.GetResult{}
		err := h.dsm.Get(&gp, &res)
		if err != nil {
			if err == repo.ErrNotFound {
				util.NotFoundHandler(w, r)
				return
			}
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		if res.Dataset == nil || res.Dataset.IsEmpty() {
			util.WriteErrResponse(w, http.StatusNotFound, errors.New("cannot find peer dataset"))
			return
		}

		// TODO (b5) - why is this necessary?
		ref := reporef.DatasetRef{
			Peername:  res.Dataset.Peername,
			ProfileID: profile.ID(res.Dataset.ProfileID),
			Name:      res.Dataset.Name,
			Path:      res.Dataset.Path,
			FSIPath:   res.Ref.FSIPath,
			Published: res.Ref.Published,
			Dataset:   res.Dataset,
		}

		util.WriteResponse(w, ref)
		return
	}
}

// WriteHandler writes input data to the local filesystem link
func (h *FSIHandlers) WriteHandler(routePrefix string) http.HandlerFunc {
	handler := h.writeHandler(routePrefix)
	return func(w http.ResponseWriter, r *http.Request) {
		if h.ReadOnly {
			readOnlyResponse(w, routePrefix)
			return
		}

		switch r.Method {
		case "OPTIONS":
			util.EmptyOkHandler(w, r)
		case "POST":
			handler(w, r)
		default:
			util.NotFoundHandler(w, r)
		}
	}
}

func (h *FSIHandlers) writeHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ref, err := DatasetRefFromPath(r.URL.Path[len(routePrefix):])
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("bad reference: %s", err.Error()))
			return
		}

		ds := &dataset.Dataset{}
		if err := json.NewDecoder(r.Body).Decode(ds); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		p := &lib.FSIWriteParams{
			Ref: ref.AliasString(),
			Ds:  ds,
		}

		out := []lib.StatusItem{}
		if err := h.Write(p, &out); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		util.WriteResponse(w, out)
	}
}

// CheckoutHandler invokes checkout via an API call
func (h *FSIHandlers) CheckoutHandler(routePrefix string) http.HandlerFunc {
	handleCheckout := h.checkoutHandler(routePrefix)
	return func(w http.ResponseWriter, r *http.Request) {
		if h.ReadOnly {
			readOnlyResponse(w, routePrefix)
			return
		}

		switch r.Method {
		case "OPTIONS":
			util.EmptyOkHandler(w, r)
		case "POST":
			handleCheckout(w, r)
		default:
			util.NotFoundHandler(w, r)
		}
	}
}

func (h *FSIHandlers) checkoutHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ref, err := DatasetRefFromPath(r.URL.Path[len(routePrefix):])
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("bad reference: %s", err.Error()))
			return
		}

		p := &lib.CheckoutParams{
			Dir: r.FormValue("dir"),
			Ref: ref.String(),
		}

		var res string
		if err := h.Checkout(p, &res); err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}

		util.WriteResponse(w, res)
	}
}

// RestoreHandler invokes restore via an API call
func (h *FSIHandlers) RestoreHandler(routePrefix string) http.HandlerFunc {
	handleRestore := h.restoreHandler(routePrefix)
	return func(w http.ResponseWriter, r *http.Request) {
		if h.ReadOnly {
			readOnlyResponse(w, routePrefix)
			return
		}

		switch r.Method {
		case "OPTIONS":
			util.EmptyOkHandler(w, r)
		case "POST":
			handleRestore(w, r)
		default:
			util.NotFoundHandler(w, r)
		}
	}
}

func (h *FSIHandlers) restoreHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ref, err := DatasetRefFromPath(r.URL.Path[len(routePrefix):])
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("bad reference: %s", err.Error()))
			return
		}

		// Add the path for the version to restore
		ref.Path = r.FormValue("path")

		p := &lib.RestoreParams{
			Dir:       r.FormValue("dir"),
			Ref:       ref.String(),
			Component: r.FormValue("component"),
		}

		var res string
		if err := h.Restore(p, &res); err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}

		util.WriteResponse(w, res)
	}
}
