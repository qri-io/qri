package api

import (
	"net/http"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/dsref"
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
	if h.readOnly {
		readOnlyResponse(w, "/push")
		return
	}

	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
		return
	case "GET":
		h.listPublicHandler(w, r)
		return
	}

	ref, err := DatasetRefFromPath(r.URL.Path[len("/push"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	p := &lib.PushParams{
		Ref:        ref.String(),
		RemoteName: r.FormValue("remote"),
	}

	var res dsref.Ref
	switch r.Method {
	case "POST":
		if err := h.Push(p, &res); err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		util.WriteResponse(w, "ok")
		return
	case "DELETE":
		if err := h.Remove(p, &res); err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		util.WriteResponse(w, "ok")
		return
	default:
		util.NotFoundHandler(w, r)
	}
}

// FeedsHandler fetches an index of named feeds
func (h *RemoteClientHandlers) FeedsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.feedsHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *RemoteClientHandlers) feedsHandler(w http.ResponseWriter, r *http.Request) {
	res := map[string][]dsref.VersionInfo{}
	remName := r.FormValue("remote")
	if err := h.Feeds(&remName, &res); err != nil {
		log.Infof("home error: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}

// DatasetPreviewHandler fetches a dataset preview from the registry
func (h *RemoteClientHandlers) DatasetPreviewHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.previewHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// TODO(dustmop): Add a test for this. Need NewTestRunnerWithMockRemote for API
func (h *RemoteClientHandlers) previewHandler(w http.ResponseWriter, r *http.Request) {
	p := &lib.PreviewParams{
		RemoteName: r.FormValue("remote"),
		Ref:        HTTPPathToQriPath(strings.TrimPrefix(r.URL.Path, "/preview/")),
	}
	res := &dataset.Dataset{}
	if err := h.Preview(p, res); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	util.WriteResponse(w, res)
}

func (h *RemoteClientHandlers) listPublicHandler(w http.ResponseWriter, r *http.Request) {
	args := lib.ListParamsFromRequest(r)
	args.OrderBy = "created"
	args.Public = true

	dsm := lib.NewDatasetMethods(h.inst)

	res := []dsref.VersionInfo{}
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
