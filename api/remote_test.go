package api

import (
	// "math/rand"
	"testing"
)

func TestRemoteClientHandlers(t *testing.T) {
	t.Skip("TODO(dlong): Skip for now, returning a 500, need to investigate")

	node, teardown := newTestNodeWithNumDatasets(t, 2)
	defer teardown()

	inst := newTestInstanceWithProfileFromNode(node)
	l := NewLogHandlers(inst)
	h := NewRemoteClientHandlers(inst, false)

	publishCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/publish/", nil},
		{"POST", "/publish/me/cities", nil},
		{"DELETE", "/publish/me/cities", nil},
	}
	runHandlerTestCases(t, "publish", h.PublishHandler, publishCases, true)

	fetchCases := []handlerTestCase{
		{"GET", "/history/", nil},
		{"GET", "/history/me/cities", nil},
	}
	runHandlerTestCases(t, "fetch", l.LogHandler, fetchCases, true)
}
