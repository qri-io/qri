package peers

import (
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/server/logging"

	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

func NewHandlers(log logging.Logger, r repo.Repo, node *p2p.QriNode) *Handlers {
	req := NewRequests(r, node)
	h := Handlers{*req, log}
	return &h
}

// Handlers wraps a requests struct to interface with http.HandlerFunc
type Handlers struct {
	Requests
	log logging.Logger
}

func (h *Handlers) PeersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.listPeersHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *Handlers) PeerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.getPeerHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *Handlers) PeerNamespaceHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.peerNamespaceHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *Handlers) ConnectToPeerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.connectToPeerHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (d *Handlers) ConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		d.listConnectionsHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *Handlers) listPeersHandler(w http.ResponseWriter, r *http.Request) {
	p := util.PageFromRequest(r)
	res := []*profile.Profile{}
	args := &ListParams{
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

func (h *Handlers) listConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	util.WriteResponse(w, h.qriNode.ConnectedPeers())
}

func (h *Handlers) connectToPeerHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *Handlers) getPeerHandler(w http.ResponseWriter, r *http.Request) {
	res := &profile.Profile{}
	args := &GetParams{
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

func (h *Handlers) peerNamespaceHandler(w http.ResponseWriter, r *http.Request) {
	page := util.PageFromRequest(r)
	args := &NamespaceParams{
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
