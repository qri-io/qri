package api

import (
	"net/http"

	"github.com/qri-io/apiutil"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
)

// RenderHandlers wraps a requests struct to interface with http.HandlerFunc
type RenderHandlers struct {
	lib.RenderRequests
	repo repo.Repo
}

// NewRenderHandlers allocates a RenderHandlers pointer
func NewRenderHandlers(r repo.Repo) *RenderHandlers {
	req := lib.NewRenderRequests(r, nil)
	h := RenderHandlers{*req, r}
	return &h
}

// RenderHandler renders a given dataset ref
func (h *RenderHandlers) RenderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		apiutil.EmptyOkHandler(w, r)
		return
	}

	p := &lib.RenderParams{
		Ref:            HTTPPathToQriPath(r.URL.Path[len("/render"):]),
		TemplateFormat: "html",
	}

	data := []byte{}
	if err := h.Render(p, &data); err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.Write(data)
}
