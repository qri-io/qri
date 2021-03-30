package api

import (
	// "math/rand"
	"context"
	"testing"

	"github.com/qri-io/qri/lib"
)

func TestRemoteClientHandlers(t *testing.T) {
	t.Skip("TODO(dlong): Skip for now, returning a 500, need to investigate")

	node, teardown := newTestNode(t)
	defer teardown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inst := newTestInstanceWithProfileFromNode(ctx, node)
	historyHandler := lib.NewHTTPRequestHandler(inst, "log.history")
	h := NewRemoteClientHandlers(inst, false)

	publishCases := []handlerTestCase{
		{"GET", "/publish/", nil, nil},
		{"POST", "/publish/me/cities", nil, nil},
		{"DELETE", "/publish/me/cities", nil, nil},
	}
	runHandlerTestCases(t, "publish", h.PushHandler, publishCases, true)

	// tests getting a list of logs from a remote
	fetchCases := []handlerTestCase{
		{"POST", "/history/", nil, nil},
		{"POST", "/history/me/cities", nil, nil},
	}
	runHandlerTestCases(t, "fetch", historyHandler, fetchCases, true)
}
