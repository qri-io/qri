package api

import (
	"testing"
)

func TestRenderHandler(t *testing.T) {
	r, teardown := newTestRepo(t)
	defer teardown()

	cases := []handlerTestCase{
		{"OPTIONS", "/render", nil},
		{"GET", "/render/me/movies", nil},
	}

	h := NewRenderHandlers(r)
	runHandlerTestCases(t, "render", h.RenderHandler, cases, false)
}
