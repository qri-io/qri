package api

import (
	"testing"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
)

func TestRegistryHandlers(t *testing.T) {
	r, teardown := newTestRepo(t)
	defer teardown()

	node, err := p2p.NewQriNode(r, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	h := NewRegistryHandlers(node)

	registryCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/registry/me/counter", nil},
		{"DELETE", "/registry/me/counter", nil},
		{"PATCH", "/", nil},
	}
	runHandlerTestCases(t, "registry", h.RegistryHandler, registryCases)
}
