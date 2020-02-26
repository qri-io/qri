package api

import (
	"encoding/json"
	"net/http"

	"github.com/qri-io/apiutil"
	"github.com/qri-io/dataset"
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
		Ref:       HTTPPathToQriPath(r.URL.Path[len("/render"):]),
		OutFormat: "html",
	}

	// support rendering a passed-in JSON dataset document
	if r.Header.Get("Content-Type") == "application/json" {
		ds := &dataset.Dataset{}
		if err := json.NewDecoder(r.Body).Decode(ds); err != nil {
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		p.Dataset = ds
	}

	// Old style viz component rendering
	if r.FormValue("viz") == "true" {
		data := []byte{}
		if err := h.RenderViz(p, &data); err != nil {
			apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}

		w.Write(data)
		return
	}

	// Readme component rendering
	p.UseFSI = r.FormValue("fsi") == "true"
	var text string
	if err := h.RenderReadme(p, &text); err != nil {
		apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	w.Write([]byte(text))
}
