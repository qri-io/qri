package api

import (
	"math/rand"
	"testing"

	"github.com/qri-io/dag/dsync"
)

func TestRemoteHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	// Set a seed so that the sessionID is deterministic
	rand.Seed(1234)

	testCases := []handlerTestCase{
		{"POST", "/", mustFile(t, "testdata/postRemoteRequest.json")},
	}

	testReceivers := dsync.NewTestReceivers()

	// Reject all dag.Info's
	// TODO (b5): this is going to break tests
	// lib.Config.API.RemoteAcceptSizeMax = 0
	rh := NewRemoteHandlers(node, nil, testReceivers)
	runHandlerTestCases(t, "remote reject", rh.ReceiveHandler, testCases, true)

	// Accept all dag.Info's
	// TODO (b5): this is going to break tests
	// lib.Config.API.RemoteAcceptSizeMax = -1
	rh = NewRemoteHandlers(node, nil, testReceivers)
	runHandlerTestCases(t, "remote accept", rh.ReceiveHandler, testCases, true)
}
