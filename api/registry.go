package api

import (
	"fmt"
	"net/http"
	"strings"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// RegistryHandlers wraps a requests struct to interface with http.HandlerFunc
type RegistryHandlers struct {
	lib.RegistryRequests
	repo repo.Repo
}

// NewRegistryHandlers allocates a RegistryHandlers pointer
func NewRegistryHandlers(node *p2p.QriNode) *RegistryHandlers {
	req := lib.NewRegistryRequests(node, nil)
	h := RegistryHandlers{*req, node.Repo}
	return &h
}

// RegistryHandler is the endpoint to call to the registry
func (h *RegistryHandlers) RegistryHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		// get a dataset if it is published, error if not
		h.registryDatasetHandler(w, r)
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

// RegistryDatasetsHandler is the endpoint to get the list of dataset summaries
// available on the registry
func (h *RegistryHandlers) RegistryDatasetsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		// returns list of published datasets on the registry
		h.listRegistryDatasetsHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// RegistryDatasetHandler is the endpoint to get a dataset summary
// from the registry
func (h *RegistryHandlers) RegistryDatasetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		// returns a registry dataset
		h.registryDatasetHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *RegistryHandlers) publishRegistryHandler(w http.ResponseWriter, r *http.Request) {
	ref, err := DatasetRefFromPath(r.URL.Path[len("/registry"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	var res bool
	if err = h.RegistryRequests.Publish(&ref, &res); err != nil {
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

func (h *RegistryHandlers) listRegistryDatasetsHandler(w http.ResponseWriter, r *http.Request) {
	args := lib.ListParamsFromRequest(r)
	params := &lib.RegistryListParams{
		Limit:  args.Limit,
		Offset: args.Offset,
	}
	var res bool
	if err := h.RegistryRequests.List(params, &res); err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, fmt.Errorf("error getting list of datasets available on the registry: %s", err))
		return
	}

	if err := util.WritePageResponse(w, params.Refs, r, args.Page()); err != nil {
		log.Infof("error listing registry datasets: %s", err.Error())
	}
}

func (h *RegistryHandlers) registryDatasetHandler(w http.ResponseWriter, r *http.Request) {
	res := &repo.DatasetRef{}
	ref, err := DatasetRefFromPath(r.URL.Path[len("/registry/"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
	if err = repo.CanonicalizeDatasetRef(h.repo, &ref); err != nil && err != repo.ErrNotFound {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	err = h.RegistryRequests.GetDataset(&ref, res)
	if err != nil {
		// If error was that the dataset wasn't found, send back 404 Not Found.
		if strings.HasPrefix(err.Error(), "error 404") {
			util.NotFoundHandler(w, r)
			return
		}
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	util.WriteResponse(w, res)
}
