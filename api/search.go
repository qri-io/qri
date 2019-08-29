package api

import (
	"encoding/json"
	"net/http"

	util "github.com/qri-io/apiutil"
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
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.searchHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *SearchHandlers) searchHandler(w http.ResponseWriter, r *http.Request) {

	listParams := lib.ListParamsFromRequest(r)
	sp := &lib.SearchParams{
		QueryString: r.FormValue("q"),
		Limit:       listParams.Limit,
		Offset:      listParams.Offset,
	}

	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(sp); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
	}

	results := []lib.SearchResult{}

	if err := h.SearchMethods.Search(sp, &results); err != nil {
		log.Infof("search error: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, results)
}
