package api

import (
	"testing"
)

func TestRegistryHandlers(t *testing.T) {
	r, teardown := newTestRepo(t)
	defer teardown()

	h := NewRegistryHandlers(r)

	registryCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"POST", "/registry/me/counter", nil},
		{"DELETE", "/registry/me/counter", nil},
		{"PATCH", "/", nil},
	}
	runHandlerTestCases(t, "registry", h.RegistryHandler, registryCases)
}
