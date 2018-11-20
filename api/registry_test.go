package api

import (
	"testing"
)

func TestRegistryHandlers(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	h := NewRegistryHandlers(node)

	registryCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/registry/me/counter", nil},
		{"DELETE", "/registry/me/counter", nil},
		{"PATCH", "/", nil},
	}
	runHandlerTestCases(t, "registry", h.RegistryHandler, registryCases)

	registryListCases := []handlerTestCase{
		{"GET", "/registry/list", nil},
	}
	runHandlerTestCases(t, "registryList", h.RegistryListHandler, registryListCases)
}
