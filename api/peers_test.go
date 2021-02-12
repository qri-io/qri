package api

import (
	"context"
	"testing"
)

func TestPeerHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := NewPeerHandlers(newTestInstanceWithProfileFromNode(ctx, node), false)

	connectionsCases := []handlerTestCase{
		{"GET", "/", nil, nil},
		{"DELETE", "/", nil, nil},
	}
	runHandlerTestCases(t, "connections", h.ConnectionsHandler, connectionsCases, true)

}
