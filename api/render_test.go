package api

import (
	"testing"
)

func TestRenderHandler(t *testing.T) {
	r, teardown := newTestRepo(t)
	defer teardown()

	cases := []handlerTestCase{
		// {"OPTIONS", "OPTIONS", "/", nil},
	}

	h := NewRenderHandlers(r)
	runHandlerTestCases(t, "render", h.RenderHandler, cases)
}
