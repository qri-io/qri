package search

import (
	"encoding/json"
	"net/http"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/server/logging"
)

func NewSearchHandlers(log logging.Logger, store cafs.Filestore, r repo.Repo) *SearchHandlers {
	req := NewSearchRequests(store, r)
	return &SearchHandlers{*req, log}
}

// SearchHandlers wraps a requests struct to interface with http.HandlerFunc
type SearchHandlers struct {
	SearchRequests
	log logging.Logger
}

func (h *SearchHandlers) SearchHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET", "POST":
		h.searchHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *SearchHandlers) searchHandler(w http.ResponseWriter, r *http.Request) {
	p := util.PageFromRequest(r)

	sp := &SearchParams{
		Query:  r.FormValue("q"),
		Limit:  p.Limit(),
		Offset: p.Offset(),
	}

	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(sp); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
	}

	res := make([]*repo.DatasetRef, p.Limit())
	if err := h.Search(sp, &res); err != nil {
		h.log.Infof("search error: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}
