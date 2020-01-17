// Package regclient defines a client for interacting with a registry server
package regclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/qri-io/dataset"
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

// HomeFeed fetches the first page of featured & recent feeds in one call
func (c *Client) HomeFeed(ctx context.Context) (map[string][]*dataset.Dataset, error) {
	if c.cfg.Location == "" {
		return nil, ErrNoRegistry
	}

	// TODO (b5) - update registry endpoint name
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/dataset_summary/splash", c.cfg.Location), nil)
	if err != nil {
		return nil, err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "no such host") {
			return nil, ErrNoRegistry
		}
		return nil, err
	}
	// add response to an envelope
	env := struct {
		Data map[string][]*dataset.Dataset
		Meta struct {
			Error  string
			Status string
			Code   int
		}
	}{}

	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error %d: %s", res.StatusCode, env.Meta.Error)
	}
	return env.Data, nil
}
