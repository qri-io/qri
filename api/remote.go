package api

import (
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/lib"
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

// PushHandler facilitates requests to push dataset data from a local node
// to a remote. It also supports remove requests to a remote for legacy reasons
func (h *RemoteClientHandlers) PushHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.listPublicHandler(w, r)
		return
	}

	if h.readOnly {
		readOnlyResponse(w, "/push")
		return
	}

	params := lib.PushParams{}
	err := lib.UnmarshalParams(r, &params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	switch r.Method {
	case http.MethodPost:
		res, err := h.Push(r.Context(), &params)
		if err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		util.WriteResponse(w, res)
		return
	case http.MethodDelete:
		res, err := h.Remove(r.Context(), &params)
		if err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		util.WriteResponse(w, res)
		return
	default:
		util.NotFoundHandler(w, r)
	}
}

// FeedsHandler fetches an index of named feeds
func (h *RemoteClientHandlers) FeedsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		h.feedsHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *RemoteClientHandlers) feedsHandler(w http.ResponseWriter, r *http.Request) {
	params := lib.FeedsParams{}
	err := lib.UnmarshalParams(r, &params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	res, err := h.Feeds(r.Context(), &params)
	if err != nil {
		log.Infof("home error: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

// DatasetPreviewHandler fetches a dataset preview from the registry
func (h *RemoteClientHandlers) DatasetPreviewHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet, http.MethodPost:
		h.previewHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// TODO(dustmop): Add a test for this. Need NewTestRunnerWithMockRemote for API
func (h *RemoteClientHandlers) previewHandler(w http.ResponseWriter, r *http.Request) {
	params := lib.PreviewParams{}
	err := lib.UnmarshalParams(r, &params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	res, err := h.Preview(r.Context(), &params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h *RemoteClientHandlers) listPublicHandler(w http.ResponseWriter, r *http.Request) {
	args := lib.ListParamsFromRequest(r)
	args.OrderBy = "created"
	args.Public = true

	res, err := h.inst.Dataset().List(r.Context(), &args)
	if err != nil {
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
