package api

import (
	"net/http"

	util "github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/lib"
)

// RenderHandlers wraps a requests struct to interface with http.HandlerFunc
type RenderHandlers struct {
	lib.RenderMethods
}

// NewRenderHandlers allocates a RenderHandlers pointer
func NewRenderHandlers(inst *lib.Instance) *RenderHandlers {
	req := lib.NewRenderMethods(inst)
	h := RenderHandlers{*req}
	return &h
}

// RenderHandler renders a given dataset ref
func (h *RenderHandlers) RenderHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.RenderParams{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	// Old style viz component rendering
	if params.Viz {
		data, err := h.RenderViz(r.Context(), params)
		if err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		w.Write(data)
		return
	}

	// Readme component rendering
	data, err := h.RenderReadme(r.Context(), params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	w.Write(data)
}
