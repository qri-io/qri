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

	// node, teardown := newTestNode(t)
	// defer teardown()

	// inst := newTestInstanceWithProfileFromNode(node)

	// // Set a seed so that the sessionID is deterministic
	// rand.Seed(1234)

	// testCases := []handlerTestCase{
	// 	{"POST", "/", mustFile(t, "testdata/postRemoteRequest.json")},
	// }

	// cfg, _ := testConfigAndSetter()
	// // testReceivers := dsync.NewTestReceivers()

	// // Reject all dag.Info's
	// cfg.API.RemoteAcceptSizeMax = 0
	// rh := NewRemoteHandlers(inst)
	// runHandlerTestCases(t, "remote reject", rh.ReceiveHandler, testCases, true)

	// // Accept all dag.Info's
	// cfg.API.RemoteAcceptSizeMax = -1
	// rh = NewRemoteHandlers(inst)
	// runHandlerTestCases(t, "remote accept", rh.ReceiveHandler, testCases, true)
}
