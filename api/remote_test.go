package api

import (
	"testing"

	"github.com/qri-io/qri/lib"
)

func TestRemoteHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	testCases := []handlerTestCase{
		{"POST", "/", mustFile(t, "testdata/postRemoteRequest.json")},
	}

	// Reject all dag.Info's
	lib.Config.API.RemoteAcceptSizeMax = 0
	rh := NewRemoteHandlers(node)
	runHandlerTestCases(t, "remote reject", rh.ReceiveHandler, testCases, true)

	// Accept all dag.Info's
	lib.Config.API.RemoteAcceptSizeMax = -1
	rh = NewRemoteHandlers(node)
	runHandlerTestCases(t, "remote accept", rh.ReceiveHandler, testCases, true)
}
