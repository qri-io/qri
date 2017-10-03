package server

import (
	"encoding/json"
	"fmt"
	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"net/http"
)

func NewSearchHandlers(store cafs.Filestore, r repo.Repo, node *p2p.QriNode) *SearchHandlers {
	req := NewSearchRequests(store, r, node)
	return &SearchHandlers{*req}
}

// SearchHandlers wraps a requests struct to interface with http.HandlerFunc
type SearchHandlers struct {
	SearchRequests
}

func (h *SearchHandlers) SearchHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "POST":
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
		fmt.Println("err:")
		fmt.Println(err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}
