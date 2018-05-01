package api

import (
	"fmt"
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// PeerHandlers wraps a requests struct to interface with http.HandlerFunc
type PeerHandlers struct {
	core.PeerRequests
	repo     repo.Repo
	ReadOnly bool
}

// NewPeerHandlers allocates a PeerHandlers pointer
func NewPeerHandlers(r repo.Repo, node *p2p.QriNode, readOnly bool) *PeerHandlers {
	req := core.NewPeerRequests(node, nil)
	h := PeerHandlers{*req, r, readOnly}
	return &h
}

// PeersHandler is the endpoint for fetching peers
func (h *PeerHandlers) PeersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		if h.ReadOnly {
			readOnlyResponse(w, "/peers")
		} else {
			h.listPeersHandler(w, r)
		}
	default:
		util.NotFoundHandler(w, r)
	}
}

// PeerHandler gets info on a single peer
func (h *PeerHandlers) PeerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		if h.ReadOnly {
			readOnlyResponse(w, "/peers/")
			return
		}
		h.peerHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// ConnectToPeerHandler is the endpoint for explicitly connecting to a peer
func (h *PeerHandlers) ConnectToPeerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.connectToPeerHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// ConnectionsHandler is the endpoint for listing qri & IPFS connections
func (h *PeerHandlers) ConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		if h.ReadOnly {
			readOnlyResponse(w, "/connections")
			return
		}
		h.listConnectionsHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *PeerHandlers) listPeersHandler(w http.ResponseWriter, r *http.Request) {
	args := core.ListParamsFromRequest(r)
	args.OrderBy = "created"
	res := []*config.ProfilePod{}
	if err := h.List(&args, &res); err != nil {
		log.Infof("list peers: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WritePageResponse(w, res, r, args.Page())
}

func (h *PeerHandlers) listConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	//limit := 0
	// TODO: double check with @b5 on this change
	listParams := core.ListParamsFromRequest(r)
	peers := []string{}

	if err := h.ConnectedIPFSPeers(&listParams.Limit, &peers); err != nil {
		log.Infof("error showing connected peers: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, peers)
}

func (h *PeerHandlers) peerHandler(w http.ResponseWriter, r *http.Request) {
	proid := r.URL.Path[len("/peers/"):]
	id, err := profile.IDB58Decode(proid)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	p := &core.PeerInfoParams{
		ProfileID: id,
	}
	res := &config.ProfilePod{}
	if err := h.Info(p, res); err != nil {
		log.Infof("error getting peer info: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h *PeerHandlers) namespaceHandler(w http.ResponseWriter, r *http.Request) {

}

func (h *PeerHandlers) connectToPeerHandler(w http.ResponseWriter, r *http.Request) {
	b58pid := r.URL.Path[len("/connect/"):]

	if b58pid == "" {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("peer id is required"))
		return
	}

	res := &config.ProfilePod{}
	if err := h.ConnectToPeer(&b58pid, res); err != nil {
		log.Infof("error connecting to peer: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}
