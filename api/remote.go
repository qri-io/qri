package api

import (
	"net/http"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/dag/dsync"
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
		DsyncHandler:  dsync.HTTPRemoteHandler(inst.Dsync()),
	}
}

// PublicationHandler facilitates requests to publish or unpublish from the local
// node to a remote
func (h *RemoteHandlers) PublicationHandler(w http.ResponseWriter, r *http.Request) {
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

// // ReceiveHandler is the endpoint for remotes to receive daginfo
// func (h *RemoteHandlers) ReceiveHandler(w http.ResponseWriter, r *http.Request) {
// 	switch r.Method {
// 	// no "OPTIONS" method here, because browsers should never hit this endpoint
// 	case "POST":
// 		h.receiveDataset(w, r)
// 	default:
// 		util.NotFoundHandler(w, r)
// 	}
// }

// // CompleteHandler is the endpoint for remotes when they complete the dsync process
// func (h *RemoteHandlers) CompleteHandler(w http.ResponseWriter, r *http.Request) {
// 	switch r.Method {
// 	// no "OPTIONS" method here, because browsers should never hit this endpoint
// 	case "POST":
// 		h.completeDataset(w, r)
// 	default:
// 		util.NotFoundHandler(w, r)
// 	}
// }

// func (h *RemoteHandlers) receiveDataset(w http.ResponseWriter, r *http.Request) {
// 	var params lib.ReceiveParams
// 	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
// 		util.WriteErrResponse(w, http.StatusInternalServerError, err)
// 		return
// 	}

// 	var result lib.ReceiveResult
// 	err := h.Receive(&params, &result)
// 	if err != nil {
// 		util.WriteErrResponse(w, http.StatusInternalServerError, err)
// 		return
// 	}
// 	if result.Success {
// 		util.WriteResponse(w, result)
// 		return
// 	}

// 	util.WriteErrResponse(w, http.StatusForbidden, fmt.Errorf("%s", result.RejectReason))
// }

// func (h *RemoteHandlers) completeDataset(w http.ResponseWriter, r *http.Request) {
// 	var params lib.CompleteParams
// 	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
// 		util.WriteErrResponse(w, http.StatusInternalServerError, err)
// 		return
// 	}

// 	var result bool
// 	err := h.Complete(&params, &result)
// 	if err != nil {
// 		util.WriteErrResponse(w, http.StatusInternalServerError, err)
// 		return
// 	}

// 	util.WriteResponse(w, "Success")
// }
