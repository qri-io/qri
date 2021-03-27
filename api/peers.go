package api

import (
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/lib"
)

// PeerHandlers wraps a requests struct to interface with http.HandlerFunc
type PeerHandlers struct {
	lib.PeerMethods
	ReadOnly bool
}

// NewPeerHandlers allocates a PeerHandlers pointer
func NewPeerHandlers(inst *lib.Instance, readOnly bool) *PeerHandlers {
	h := PeerHandlers{inst.Peer(), readOnly}
	return &h
}

// PeersHandler is the endpoint for fetching peers
func (h *PeerHandlers) PeersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
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
	case http.MethodGet:
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
	case http.MethodPost:
		h.connectToPeerHandler(w, r)
	case http.MethodDelete:
		h.disconnectFromPeerHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// ConnectionsHandler is the endpoint for listing qri & IPFS connections
func (h *PeerHandlers) ConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if h.ReadOnly {
			readOnlyResponse(w, "/connections")
			return
		}
		h.listConnectionsHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// QriConnectionsHandler is the endpoint for listing qri profile connections
func (h *PeerHandlers) QriConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if h.ReadOnly {
			readOnlyResponse(w, "/connections/qri")
			return
		}
		h.listQriConnectionsHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *PeerHandlers) listPeersHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.PeerListParams{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.List(r.Context(), params)
	if err != nil {
		log.Infof("list peers: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	page := util.NewPageFromOffsetAndLimit(params.Offset, params.Limit)
	util.WritePageResponse(w, res, r, page)
}

func (h *PeerHandlers) listConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO(arqu): we don't utilize limit, but should
	params := &lib.ConnectionsParams{Limit: -1}
	peers, err := h.Connections(r.Context(), params)
	if err != nil {
		log.Infof("error showing connected peers: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, peers)
}

func (h *PeerHandlers) listQriConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.ConnectionsParams{Limit: util.ReqParamInt(r, "limit", 25)}
	peers, err := h.ConnectedQriProfiles(r.Context(), params)
	if err != nil {
		log.Infof("error showing connected qri profiles: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, peers)
}

func (h *PeerHandlers) peerHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.PeerInfoParams{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.Info(r.Context(), params)
	if err != nil {
		log.Infof("error getting peer info: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h *PeerHandlers) connectToPeerHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.ConnectParamsPod{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.Connect(r.Context(), params)
	if err != nil {
		log.Infof("error connecting to peer: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h *PeerHandlers) disconnectFromPeerHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.ConnectParamsPod{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	if err := h.Disconnect(r.Context(), params); err != nil {
		log.Infof("error connecting to peer: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, nil)
}
