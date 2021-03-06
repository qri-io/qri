package api

import (
	"net/http"
	"strings"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
)

// FSIHandlers connects HTTP requests to the FSI subsystem
type FSIHandlers struct {
	inst     *lib.Instance
	dsm      *lib.DatasetMethods
	ReadOnly bool
}

// NewFSIHandlers creates handlers that talk to qri's filesystem integration
func NewFSIHandlers(inst *lib.Instance, readOnly bool) FSIHandlers {
	return FSIHandlers{
		inst:     inst,
		dsm:      lib.NewDatasetMethods(inst),
		ReadOnly: readOnly,
	}
}

// CanInitDatasetWorkDirHandler returns whether a directory can be initialized
func (h *FSIHandlers) CanInitDatasetWorkDirHandler(routePrefix string) http.HandlerFunc {
	handleCanInit := h.canInitDatasetWorkDirHandler(routePrefix)

	return func(w http.ResponseWriter, r *http.Request) {
		if h.ReadOnly {
			readOnlyResponse(w, "/caninitdatasetworkdir")
			return
		}

		switch r.Method {
		case http.MethodPost:
			handleCanInit(w, r)
		default:
			util.NotFoundHandler(w, r)
		}
	}
}

func (h *FSIHandlers) canInitDatasetWorkDirHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		method := "fsi.caninitdatasetworkdir"
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

// WriteHandler writes input data to the local filesystem link
func (h *FSIHandlers) WriteHandler(routePrefix string) http.HandlerFunc {
	handler := h.writeHandler(routePrefix)
	return func(w http.ResponseWriter, r *http.Request) {
		if h.ReadOnly {
			readOnlyResponse(w, routePrefix)
			return
		}

		switch r.Method {
		case http.MethodPost:
			handler(w, r)
		default:
			util.NotFoundHandler(w, r)
		}
	}
}

func (h *FSIHandlers) writeHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		method := "fsi.write"
		p := h.inst.NewInputParam(method)

		// TODO(dustmop): Add this to lib.UnmarshalParams for methods that can
		// receive a refstr in the URL, or annotate the param struct with
		// a tag and marshal the url to that field
		err := addDsRefFromURL(r, routePrefix)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

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

// CreateLinkHandler creates an fsi link
func (h *FSIHandlers) CreateLinkHandler(routePrefix string) http.HandlerFunc {
	handler := h.createLinkHandler(routePrefix)
	return func(w http.ResponseWriter, r *http.Request) {
		if h.ReadOnly {
			readOnlyResponse(w, routePrefix)
			return
		}

		switch r.Method {
		case http.MethodPost:
			handler(w, r)
		default:
			util.NotFoundHandler(w, r)
		}
	}
}

func (h *FSIHandlers) createLinkHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		method := "fsi.createlink"
		p := h.inst.NewInputParam(method)

		// TODO(dustmop): Add this to lib.UnmarshalParams for methods that can
		// receive a refstr in the URL, or annotate the param struct with
		// a tag and marshal the url to that field
		err := addDsRefFromURL(r, routePrefix)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

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

// UnlinkHandler unlinks a working directory
func (h *FSIHandlers) UnlinkHandler(routePrefix string) http.HandlerFunc {
	handler := h.unlinkHandler(routePrefix)
	return func(w http.ResponseWriter, r *http.Request) {
		if h.ReadOnly {
			readOnlyResponse(w, routePrefix)
			return
		}

		switch r.Method {
		case http.MethodPost:
			handler(w, r)
		default:
			util.NotFoundHandler(w, r)
		}
	}
}

func (h *FSIHandlers) unlinkHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		method := "fsi.unlink"
		p := h.inst.NewInputParam(method)

		// TODO(dustmop): Add this to lib.UnmarshalParams for methods that can
		// receive a refstr in the URL, or annotate the param struct with
		// a tag and marshal the url to that field
		err := addDsRefFromURL(r, routePrefix)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

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

// CheckoutHandler invokes checkout via an API call
func (h *FSIHandlers) CheckoutHandler(routePrefix string) http.HandlerFunc {
	handleCheckout := h.checkoutHandler(routePrefix)
	return func(w http.ResponseWriter, r *http.Request) {
		if h.ReadOnly {
			readOnlyResponse(w, routePrefix)
			return
		}

		switch r.Method {
		case http.MethodPost:
			handleCheckout(w, r)
		default:
			util.NotFoundHandler(w, r)
		}
	}
}

func (h *FSIHandlers) checkoutHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		method := "fsi.checkout"
		p := h.inst.NewInputParam(method)

		// TODO(dustmop): Add this to lib.UnmarshalParams for methods that can
		// receive a refstr in the URL, or annotate the param struct with
		// a tag and marshal the url to that field
		err := addDsRefFromURL(r, routePrefix)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

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

// RestoreHandler invokes restore via an API call
func (h *FSIHandlers) RestoreHandler(routePrefix string) http.HandlerFunc {
	handleRestore := h.restoreHandler(routePrefix)
	return func(w http.ResponseWriter, r *http.Request) {
		if h.ReadOnly {
			readOnlyResponse(w, routePrefix)
			return
		}

		switch r.Method {
		case http.MethodPost:
			handleRestore(w, r)
		default:
			util.NotFoundHandler(w, r)
		}
	}
}

func (h *FSIHandlers) restoreHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		method := "fsi.restore"
		p := h.inst.NewInputParam(method)

		// TODO(dustmop): Add this to lib.UnmarshalParams for methods that can
		// receive a refstr in the URL, or annotate the param struct with
		// a tag and marshal the url to that field
		err := addDsRefFromURL(r, routePrefix)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

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

// If the route has a dataset reference in the url, parse that ref, and
// add it to the request object using the field "refstr".
func addDsRefFromURL(r *http.Request, routePrefix string) error {
	// routePrefix looks like "/route/{path:.*}" and we only want "/route/"
	pos := strings.LastIndex(routePrefix, "/")
	if pos > 1 {
		routePrefix = routePrefix[:pos+1]
	}

	// Parse the ref, then reencode it and attach back on the url
	url := r.URL.Path[len(routePrefix):]
	ref, err := lib.DsRefFromPath(url)
	if err != nil {
		if err == dsref.ErrEmptyRef {
			return nil
		}
		return err
	}
	q := r.URL.Query()
	q.Add("refstr", ref.String())
	r.URL.RawQuery = q.Encode()
	return nil
}
