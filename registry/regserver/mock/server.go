// Package mock provides a mock registry server for testing purposes
// it mocks the behaviour of a registry server with in-memory storage
package mock

import (
	"net/http/httptest"

	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regclient"
	"github.com/qri-io/qri/registry/regserver/handlers"
)

func init() {
	// don't need verbose logging when working with mock servers
	handlers.SetLogLevel("error")
}

// NewMockServer creates an in-memory mock server (with a pinset) without any access protection and
// a registry client to match
func NewMockServer() (*regclient.Client, *httptest.Server) {
	return NewMockServerRegistry(NewMemRegistry())
}

// NewMockServerRegistry creates a mock server & client with a passed-in registry
func NewMockServerRegistry(reg registry.Registry) (*regclient.Client, *httptest.Server) {
	s := httptest.NewServer(handlers.NewRoutes(reg))
	c := regclient.NewClient(&regclient.Config{Location: s.URL})
	return c, s
}

// NewMemRegistry creates a new in-memory registry
func NewMemRegistry() registry.Registry {
	return registry.Registry{
		Profiles: registry.NewMemProfiles(),
	}
}
