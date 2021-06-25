// Package regclient defines a client for interacting with a registry server
package regclient

import (
	"errors"
	"net/http"

	golog "github.com/ipfs/go-log"
)

var (
	// ErrNoRegistry indicates that no registry has been specified
	// all client methods MUST return ErrNoRegistry for all method calls
	// when config.Registry.Location is an empty string
	ErrNoRegistry = errors.New("registry: no registry specified")
	// ErrNoConnection indicates no path to the registry can be found
	ErrNoConnection = errors.New("registry: no connection")
	// ErrNotRegistered indicates this client is not registered
	ErrNotRegistered = errors.New("registry: not registered")

	// HTTPClient is hoisted here in case you'd like to use a different client instance
	// by default we just use http.DefaultClient
	HTTPClient = http.DefaultClient

	log = golog.Logger("registry")
)

// Client wraps a registry configuration with methods for interacting
// with the configured registry
type Client struct {
	cfg        *Config
	httpClient *http.Client
}

// Config encapsulates options for working with a registry
type Config struct {
	// Location is the URL base to call to
	Location string
}

// NewClient creates a registry from a provided Registry configuration
func NewClient(cfg *Config) *Client {
	return &Client{cfg, HTTPClient}
}
