package api

import (
	"errors"
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
)

// RootHandler bundles handlers that may need to be called
// by "/"
// TODO - This will be removed when we add a proper router
type RootHandler struct {
	dsh *DatasetHandlers
	ph  *PeerHandlers
}

// NewRootHandler creates a new RootHandler
func NewRootHandler(dsh *DatasetHandlers, ph *PeerHandlers) *RootHandler {
	return &RootHandler{dsh, ph}
}

// Handler is the only Handler func for the RootHandler struct
func (mh *RootHandler) Handler(w http.ResponseWriter, r *http.Request) {
	ref := DatasetRefFromCtx(r.Context())
	if ref.IsEmpty() {
		HealthCheckHandler(w, r)
		return
	}

	if ref.IsPeerRef() {
		p := &lib.PeerInfoParams{
			Peername: ref.Peername,
		}
		res := &config.ProfilePod{}
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

	p := lib.LookupParams{
		Ref: &ref,
	}
	res := lib.LookupResult{}
	err := mh.dsh.Get(&p, &res)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	if res.Data.IsEmpty() {
		util.WriteErrResponse(w, http.StatusNotFound, errors.New("cannot find peer dataset"))
		return
	}
	util.WriteResponse(w, res.Data)
	return
}
