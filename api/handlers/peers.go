package handlers

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

func NewPeerHandlers(log logging.Logger, r repo.Repo, node *p2p.QriNode) *PeerHandlers {
	req := core.NewPeerRequests(r, node)
	h := PeerHandlers{*req, log}
	return &h
}

// PeerHandlers wraps a requests struct to interface with http.HandlerFunc
type PeerHandlers struct {
	core.PeerRequests
	log logging.Logger
}

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

func (h *PeerHandlers) PeerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.getPeerHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *PeerHandlers) PeerNamespaceHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.peerNamespaceHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

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

func (d *PeerHandlers) ConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		d.listConnectionsHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *PeerHandlers) listPeersHandler(w http.ResponseWriter, r *http.Request) {
	p := util.PageFromRequest(r)
	res := []*profile.Profile{}
	args := &core.ListParams{
		Limit:   p.Limit(),
		Offset:  p.Offset(),
		OrderBy: "created",
	}
	if err := h.List(args, &res); err != nil {
		h.log.Infof("list peers: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WritePageResponse(w, res, r, p)
}

func (h *PeerHandlers) listConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	util.WriteResponse(w, h.qriNode.ConnectedPeers())
}

func (h *PeerHandlers) connectToPeerHandler(w http.ResponseWriter, r *http.Request) {
	b58pid := r.URL.Path[len("/connect/"):]
	pid, err := peer.IDB58Decode(b58pid)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	if err := h.qriNode.ConnectToPeer(pid); err != nil {
		h.log.Infof("connecting to peer: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	profile, err := h.qriNode.Repo.Peers().GetPeer(pid)
	if err != nil {
		h.log.Infof("error getting peer profile: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, profile)
}

func (h *PeerHandlers) getPeerHandler(w http.ResponseWriter, r *http.Request) {
	res := &profile.Profile{}
	args := &core.GetParams{
		Hash:     r.URL.Path[len("/peers/"):],
		Username: r.FormValue("username"),
	}
	err := h.Get(args, res)
	if err != nil {
		h.log.Infof("error getting peer profile: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}

func (h *PeerHandlers) peerNamespaceHandler(w http.ResponseWriter, r *http.Request) {
	page := util.PageFromRequest(r)
	args := &core.NamespaceParams{
		PeerId: r.URL.Path[len("/peernamespace/"):],
		Limit:  page.Limit(),
		Offset: page.Offset(),
	}
	res := []*repo.DatasetRef{}
	if err := h.GetNamespace(args, &res); err != nil {
		h.log.Infof("error getting peer namespace: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}
