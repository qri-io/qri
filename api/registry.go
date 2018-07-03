package api

import (
	"fmt"
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
)

// RegistryHandlers wraps a requests struct to interface with http.HandlerFunc
type RegistryHandlers struct {
	lib.RegistryRequests
}

// NewRegistryHandlers allocates a RegistryHandlers pointer
func NewRegistryHandlers(r repo.Repo) *RegistryHandlers {
	req := lib.NewRegistryRequests(r, nil)
	h := RegistryHandlers{*req}
	return &h
}

// RegistryHandler is the endpoint to call to the registry
func (h *RegistryHandlers) RegistryHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	// case "GET":
	// 	// get status of dataset, is it published or not
	// 	h.statusRegistryHandler(w, r)
	case "POST", "PUT":
		// publish a dataset to the registry
		h.publishRegistryHandler(w, r)
	case "DELETE":
		// unpublish a dataset from the registry
		h.unpublishRegistryHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *RegistryHandlers) statusRegistryHandler(w http.ResponseWriter, r *http.Request) {
	ref, err := DatasetRefFromPath(r.URL.Path[len("/registry"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	var res bool
	if err := h.RegistryRequests.Status(&ref, &res); err != nil {
		util.WriteResponse(w, fmt.Sprintf("error getting status from registry: %s", err))
		return
	}

	util.WriteResponse(w, fmt.Sprintf("dataset %s is published to the registry", ref))
}

func (h *RegistryHandlers) publishRegistryHandler(w http.ResponseWriter, r *http.Request) {
	ref, err := DatasetRefFromPath(r.URL.Path[len("/registry"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	var res bool
	p := &lib.PublishParams{
		Ref: ref,
		Pin: true,
	}
	if err = h.RegistryRequests.Publish(p, &res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, fmt.Sprintf("published dataset %s", ref))
}

func (h *RegistryHandlers) unpublishRegistryHandler(w http.ResponseWriter, r *http.Request) {
	ref, err := DatasetRefFromPath(r.URL.Path[len("/registry"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	var res bool
	if err = h.RegistryRequests.Unpublish(&ref, &res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, fmt.Sprintf("unpublished dataset %s", ref))
}
