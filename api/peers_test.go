package api

import (
	"testing"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
)

func TestPeerHandlers(t *testing.T) {
	r, teardown := newTestRepo(t)
	defer teardown()

	node, err := p2p.NewQriNode(r, func(o *config.P2P) {
		o.Enabled = false
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	h := NewPeerHandlers(r, node, false)

	connectionsCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/", nil},
		{"DELETE", "/", nil},
	}
	runHandlerTestCases(t, "connections", h.ConnectionsHandler, connectionsCases)

}
