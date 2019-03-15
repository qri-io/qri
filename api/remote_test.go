package api

import (
	"testing"
)

func TestRemoteHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	testCases := []handlerTestCase{
		{"POST", "/", mustFile(t, "testdata/postRemoteRequest.json")},
	}

	rh := NewRemoteHandlers(node)
	runHandlerTestCases(t, "remote", rh.ReceiveHandler, testCases, true)
}
