// Package regclient defines a client for interacting with a registry server
package regclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/qri-io/qri/dsref"
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

// assert at compile time that Client is a dsref.Resolver
var _ dsref.Resolver = (*Client)(nil)

// ResolveRef finds the identifier & HEAD path for a dataset reference
// implements dsref.Resolver interface
func (c *Client) ResolveRef(ctx context.Context, ref *dsref.Ref) (string, error) {
	if c == nil {
		return "", dsref.ErrNotFound
	}

	// TODO (b5) - for now we're just using "registry" as the returned source value
	// value should be a /dnsaddr multiaddress
	addr := "registry"

	// TODO (b5) - need to document this endpoint
	u, err := url.Parse(fmt.Sprintf("%s/remote/refs", c.cfg.Location))
	if err != nil {
		return addr, err
	}

	q := u.Query()
	q.Set("peername", ref.Username)
	q.Set("name", ref.Name)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return addr, err
	}

	req = req.WithContext(ctx)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return addr, err
	}

	if res.StatusCode == http.StatusNotFound {
		return "", dsref.ErrNotFound
	} else if res.StatusCode != http.StatusOK {
		errMsg, _ := ioutil.ReadAll(res.Body)
		return addr, fmt.Errorf("resolving dataset ref from registry failed: %s", string(errMsg))
	}

	if err := json.NewDecoder(res.Body).Decode(ref); err != nil {
		return addr, err
	}

	return addr, nil
}
