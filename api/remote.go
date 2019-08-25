package api

import (
	"net/http"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/qri/lib"
)

// RemoteHandlers wraps a request struct to interface with http.HandlerFunc
type RemoteHandlers struct {
	*lib.RemoteMethods
	DsyncHandler http.HandlerFunc
}

// NewRemoteHandlers allocates a RemoteHandlers pointer
func NewRemoteHandlers(inst *lib.Instance) *RemoteHandlers {
	return &RemoteHandlers{
		RemoteMethods: lib.NewRemoteMethods(inst),
		DsyncHandler:  inst.Remote().DsyncHTTPHandler(),
	}
}

// PublicationRequestsHandler facilitates requests to publish or unpublish
// from the local node to a remote
func (h *RemoteHandlers) PublicationRequestsHandler(w http.ResponseWriter, r *http.Request) {
	p := &lib.PublicationParams{
		Ref:        r.FormValue("ref"),
		RemoteName: r.FormValue("remote"),
	}
	var res bool

	switch r.Method {
	case "POST":
		if err := h.Publish(p, &res); err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		util.WriteResponse(w, "ok")
		return
	case "DELETE":
		if err := h.Unpublish(p, &res); err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}

		util.WriteResponse(w, "ok")
		return
	default:
		util.NotFoundHandler(w, r)
	}
}

// DatasetsHandler handles requests to a remote that change publication
// status
func (h *RemoteHandlers) DatasetsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO (b5) -
	switch r.Method {
	default:
		util.NotFoundHandler(w, r)
	}
}
