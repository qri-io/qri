package api

import (
	"fmt"
	"io/ioutil"
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
)

// RemoteHandlers wraps a request struct to interface with http.HandlerFunc
type RemoteHandlers struct {
	lib.RemoteRequests
}

// NewRemoteHandlers allocates a RemoteHandlers pointer
func NewRemoteHandlers(node *p2p.QriNode) *RemoteHandlers {
	req := lib.NewRemoteRequests(node, nil)
	return &RemoteHandlers{*req}
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

func (h *RemoteHandlers) receiveDataset(w http.ResponseWriter, r *http.Request) {
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	var rejectReason string
	params := lib.ReceiveParams{Body: string(content)}
	err = h.Receive(&params, &rejectReason)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if rejectReason == "" {
		util.WriteResponse(w, "accepted")
		return
	}

	util.WriteErrResponse(w, http.StatusForbidden, fmt.Errorf("%s", rejectReason))
}
