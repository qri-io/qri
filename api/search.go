package api

import (
	"encoding/json"
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
)

// SearchHandlers wraps a requests struct to interface with http.HandlerFunc
type SearchHandlers struct {
	lib.SearchRequests
}

// NewSearchHandlers allocates a SearchHandlers pointer
func NewSearchHandlers(node *p2p.QriNode) *SearchHandlers {
	req := lib.NewSearchRequests(node, nil)
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
	// p := util.PageFromRequest(r)
	sp := &lib.SearchParams{
		QueryString: r.FormValue("q"),
		Limit:       100,
		Offset:      0,
	}

	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(sp); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
	}

	results := []lib.SearchResult{}

	if err := h.SearchRequests.Search(sp, &results); err != nil {
		log.Infof("search error: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, results)
}
