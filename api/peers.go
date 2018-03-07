package api

import (
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/logging"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"

	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// PeerHandlers wraps a requests struct to interface with http.HandlerFunc
type PeerHandlers struct {
	core.PeerRequests
	repo repo.Repo
	log  logging.Logger
}

// NewPeerHandlers allocates a PeerHandlers pointer
func NewPeerHandlers(log logging.Logger, r repo.Repo, node *p2p.QriNode) *PeerHandlers {
	req := core.NewPeerRequests(node, nil)
	h := PeerHandlers{*req, r, log}
	return &h
}

// PeersHandler is the endpoint for fetching peers
func (h *PeerHandlers) PeersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.listPeersHandler(w, r)
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
		h.listConnectionsHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *PeerHandlers) listPeersHandler(w http.ResponseWriter, r *http.Request) {
	args := core.ListParamsFromRequest(r)
	args.OrderBy = "created"
	res := []*profile.Profile{}
	if err := h.List(&args, &res); err != nil {
		h.log.Infof("list peers: %s", err.Error())
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
		h.log.Infof("error showing connected peers: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, peers)
}

func (h *PeerHandlers) peerHandler(w http.ResponseWriter, r *http.Request) {
	p := &core.PeerInfoParams{
		PeerID: r.URL.Path[len("/peers/"):],
	}
	res := &profile.Profile{}
	if err := h.Info(p, res); err != nil {
		h.log.Infof("error getting peer info: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h *PeerHandlers) namespaceHandler(w http.ResponseWriter, r *http.Request) {

}

func (h *PeerHandlers) connectToPeerHandler(w http.ResponseWriter, r *http.Request) {
	b58pid := r.URL.Path[len("/connect/"):]

	pid, err := peer.IDB58Decode(b58pid)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res := &profile.Profile{}
	if err := h.ConnectToPeer(&pid, res); err != nil {
		h.log.Infof("error connecting to peer: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}
