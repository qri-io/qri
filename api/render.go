package api

import (
	"net/http"

	"github.com/datatogether/api/apiutil"
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

	p := &lib.RenderParams{
		Ref:            HTTPPathToQriPath(r.URL.Path[len("/render"):]),
		TemplateFormat: "html",
		// TODO - parameterize
		All:    true,
		Limit:  0,
		Offset: 0,
	}

	data := []byte{}
	if err := h.Render(p, &data); err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.Write(data)
}
