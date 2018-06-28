package api

import (
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
	case "GET":
		// get status of dataset, is it published or not
		h.checkRegistryHandler(w, r)
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

func (h *RegistryHandlers) checkRegistryHandler(w http.ResponseWriter, r *http.Request) {
	util.WriteResponse(w, "check registry handler response")
}

func (h *RegistryHandlers) publishRegistryHandler(w http.ResponseWriter, r *http.Request) {
	util.WriteResponse(w, "publish registry handler response")
}

func (h *RegistryHandlers) unpublishRegistryHandler(w http.ResponseWriter, r *http.Request) {
	util.WriteResponse(w, "unpublish registry handler response")
}
