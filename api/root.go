package api

import (
	"errors"
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// RootHandler bundles handlers that may need to be called
// by "/"
// TODO - This will be removed when we add a proper router
type RootHandler struct {
	dsh *DatasetHandlers
	ph  *PeerHandlers
}

// WebappHandler renders the home page
func (s *Server) WebappHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(s.cfg, w, "webapp")
}

// NewRootHandler creates a new RootHandler
func NewRootHandler(dsh *DatasetHandlers, ph *PeerHandlers) *RootHandler {
	return &RootHandler{dsh, ph}
}

// Handler is the only Handler func for the RootHandler struct
func (mh *RootHandler) Handler(w http.ResponseWriter, r *http.Request) {
	ref := DatasetRefFromCtx(r.Context())
	if ref.IsEmpty() {
		util.WriteErrResponse(w, http.StatusBadRequest, errors.New("invalid endpoint"))
		return
	}

	if ref.IsPeerRef() {
		p := &core.PeerInfoParams{
			Peername: ref.Peername,
		}
		res := &profile.Profile{}
		err := mh.ph.Info(p, res)
		if err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		if res.ID == "" {
			util.WriteErrResponse(w, http.StatusNotFound, errors.New("cannot find peer"))
			return
		}
		util.WriteResponse(w, res)
		return
	}

	res := repo.DatasetRef{}
	err := mh.dsh.Get(&ref, &res)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if res.Name == "" {
		util.WriteErrResponse(w, http.StatusNotFound, errors.New("cannot find peer dataset"))
		return
	}
	if res.IsEmpty() {
		util.WriteErrResponse(w, http.StatusNotFound, errors.New("cannot find peer dataset"))
		return
	}
	util.WriteResponse(w, res)
	return
}
