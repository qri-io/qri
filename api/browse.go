package api

import (
	"net/http"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/lib"
)

// BrowseHandlers provides HTTP API handlers for fetching content
// from the qri network
type BrowseHandlers struct {
	lib.BrowseMethods
}

// NewBrowseHandlers allocates a BrowseHandlers pointer
func NewBrowseHandlers(inst *lib.Instance) *BrowseHandlers {
	req := lib.NewBrowseMethods(inst)
	return &BrowseHandlers{*req}
}

// HomeHandler shows popular content across the network
func (h *BrowseHandlers) HomeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.homeHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *BrowseHandlers) homeHandler(w http.ResponseWriter, r *http.Request) {
	res := map[string][]*dataset.Dataset{}
	p := false
	if err := h.Home(&p, &res); err != nil {
		log.Infof("home error: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WriteResponse(w, res)
}
