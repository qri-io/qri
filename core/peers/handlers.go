package peers

import (
	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"net/http"
)

func NewHandlers(r repo.Repo) *Handlers {
	req := NewRequests(r)
	h := Handlers{*req}
	return &h
}

// Handlers wraps a requests struct to interface with http.HandlerFunc
type Handlers struct {
	Requests
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

func (d *Handlers) listPeersHandler(w http.ResponseWriter, r *http.Request) {
	p := util.PageFromRequest(r)
	res := []*profile.Profile{}
	args := &ListParams{
		Limit:   p.Limit(),
		Offset:  p.Offset(),
		OrderBy: "created",
	}
	if err := d.List(args, &res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WritePageResponse(w, res, r, p)
}

func (h *Handlers) getPeerHandler(w http.ResponseWriter, r *http.Request) {
	res := &profile.Profile{}
	args := &GetParams{
		Hash:     r.URL.Path[len("/peers/"):],
		Username: r.FormValue("username"),
	}
	err := h.Get(args, res)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}
