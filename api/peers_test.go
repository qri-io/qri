package api

import (
	"testing"
)

func TestPeerHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	h := NewPeerHandlers(newTestInstanceWithProfileFromNode(node), false)

	connectionsCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "connections", h.ConnectionsHandler, connectionsCases, true)

}
