package api

import (
	// "math/rand"
	"testing"
)

func TestRemoteHandlers(t *testing.T) {
	t.Skip("TODO (b5) - restore tests")
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
