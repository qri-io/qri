package api

import (
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/lib"
)

// SearchHandlers wraps a requests struct to interface with http.HandlerFunc
type SearchHandlers struct {
	Instance *lib.Instance
}

// NewSearchHandlers allocates a SearchHandlers pointer
func NewSearchHandlers(inst *lib.Instance) *SearchHandlers {
	return &SearchHandlers{Instance: inst}
}

// SearchHandler is the endpoint for searching qri
func (h *SearchHandlers) SearchHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		h.searchHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *SearchHandlers) searchHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.SearchParams{}
	if err := lib.UnmarshalParams(r, params); err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	got, _, err := h.Instance.Dispatch(r.Context(), "search.search", params)
	if err != nil {
		util.RespondWithError(w, err)
		return
	}
	res, ok := got.([]lib.SearchResult)
	if !ok {
		util.RespondWithDispatchTypeError(w, got)
		return
	}
	util.WriteResponse(w, res)
}
