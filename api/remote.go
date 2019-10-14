package api

import (
	"net/http"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
)

// RemoteClientHandlers provides HTTP handlers for issuing requests to remotes
type RemoteClientHandlers struct {
	readOnly bool
	inst     *lib.Instance
	*lib.RemoteMethods
}

// NewRemoteClientHandlers creates remote client Handlers from a qri instance
func NewRemoteClientHandlers(inst *lib.Instance, readOnly bool) *RemoteClientHandlers {
	return &RemoteClientHandlers{
		readOnly:      readOnly,
		inst:          inst,
		RemoteMethods: lib.NewRemoteMethods(inst),
	}
}

// NewFetchHandler returns an HTTP handler for fetching details from a remote
func (h *RemoteClientHandlers) NewFetchHandler(prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.readOnly {
			readOnlyResponse(w, prefix)
			return
		}

		// ref, err := DatasetRefFromPath(r.URL.Path[len(prefix):])
		// if err != nil {
		// 	util.WriteErrResponse(w, http.StatusBadRequest, err)
		// 	return
		// }
	}
}

// PublishHandler facilitates requests to publish or unpublish
// from the local node to a remote
func (h *RemoteClientHandlers) PublishHandler(w http.ResponseWriter, r *http.Request) {
	if h.readOnly {
		readOnlyResponse(w, "/publish")
		return
	}

	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
		return
	case "GET":
		h.listPublishedHandler(w, r)
		return
	}

	ref, err := DatasetRefFromPath(r.URL.Path[len("/publish"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	p := &lib.PublicationParams{
		Ref:        ref.String(),
		RemoteName: r.FormValue("remote"),
	}
	var res repo.DatasetRef

	switch r.Method {
	case "POST":
		if err := h.Publish(p, &res); err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		util.WriteResponse(w, "ok")
		return
	case "DELETE":
		if err := h.Unpublish(p, &res); err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		util.WriteResponse(w, "ok")
		return
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *RemoteClientHandlers) listPublishedHandler(w http.ResponseWriter, r *http.Request) {
	args := lib.ListParamsFromRequest(r)
	args.OrderBy = "created"
	args.Published = true

	dsm := lib.NewDatasetRequestsInstance(h.inst)

	res := []repo.DatasetRef{}
	if err := dsm.List(&args, &res); err != nil {
		log.Infof("error listing datasets: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := util.WritePageResponse(w, res, r, args.Page()); err != nil {
		log.Infof("error list datasests response: %s", err.Error())
	}
}

// RemoteHandlers wraps a request struct to interface with http.HandlerFunc
type RemoteHandlers struct {
	*lib.RemoteMethods
	DsyncHandler   http.HandlerFunc
	RefsHandler    http.HandlerFunc
	LogsyncHandler http.HandlerFunc
}

// NewRemoteHandlers allocates a RemoteHandlers pointer
func NewRemoteHandlers(inst *lib.Instance) *RemoteHandlers {
	return &RemoteHandlers{
		RemoteMethods:  lib.NewRemoteMethods(inst),
		DsyncHandler:   inst.Remote().DsyncHTTPHandler(),
		RefsHandler:    inst.Remote().RefsHTTPHandler(),
		LogsyncHandler: inst.Remote().LogsyncHTTPHandler(),
	}
}
