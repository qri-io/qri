package api

import (
	// "math/rand"
	"testing"
)

func TestRemoteClientHandlers(t *testing.T) {
	node, teardown := newTestNodeWithNumDatasets(t, 2)
	defer teardown()

	inst := newTestInstanceWithProfileFromNode(node)
	h := NewRemoteClientHandlers(inst, false)

	publishCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/publish/", nil},
		{"POST", "/publish/me/cities", nil},
		{"DELETE", "/publish/me/cities", nil},
	}
	runHandlerTestCases(t, "publish", h.PublishHandler, publishCases, true)

	fetchCases := []handlerTestCase{
		{"GET", "/fetch/", nil},
		{"GET", "/fetch/me/cities", nil},
	}
	runHandlerTestCases(t, "fetch", h.NewFetchHandler("/fetch"), fetchCases, true)
}
