package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/dag/dsync"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
)

// RemoteHandlers wraps a request struct to interface with http.HandlerFunc
type RemoteHandlers struct {
	*lib.RemoteRequests
}

// NewRemoteHandlers allocates a RemoteHandlers pointer
func NewRemoteHandlers(node *p2p.QriNode, cfg *config.Config, rec *dsync.Receivers) *RemoteHandlers {
	req := lib.NewRemoteRequests(node, cfg, nil)
	req.Receivers = rec
	return &RemoteHandlers{req}
}

// ReceiveHandler is the endpoint for remotes to receive daginfo
func (h *RemoteHandlers) ReceiveHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	// no "OPTIONS" method here, because browsers should never hit this endpoint
	case "POST":
		h.receiveDataset(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// CompleteHandler is the endpoint for remotes when they complete the dsync process
func (h *RemoteHandlers) CompleteHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	// no "OPTIONS" method here, because browsers should never hit this endpoint
	case "POST":
		h.completeDataset(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *RemoteHandlers) receiveDataset(w http.ResponseWriter, r *http.Request) {
	var params lib.ReceiveParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	var result lib.ReceiveResult
	err := h.Receive(&params, &result)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if result.Success {
		util.WriteResponse(w, result)
		return
	}

	util.WriteErrResponse(w, http.StatusForbidden, fmt.Errorf("%s", result.RejectReason))
}

func (h *RemoteHandlers) completeDataset(w http.ResponseWriter, r *http.Request) {
	var params lib.CompleteParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	var result bool
	err := h.Complete(&params, &result)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, "Success")
}
