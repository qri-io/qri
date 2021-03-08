package api

import (
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/lib"
)

// SearchHandlers wraps a requests struct to interface with http.HandlerFunc
type SearchHandlers struct {
	lib.SearchMethods
}

// NewSearchHandlers allocates a SearchHandlers pointer
func NewSearchHandlers(inst *lib.Instance) *SearchHandlers {
	req := lib.NewSearchMethods(inst)
	return &SearchHandlers{*req}
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

	res, err := h.SearchMethods.Search(r.Context(), params)
	if err != nil {
		log.Infof("search error: %s", err.Error())
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	util.WriteResponse(w, res)
}
