package regclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/qri/registry"
)

// GetProfile fills in missing fields in p with registry data
func (c Client) GetProfile(p *registry.Profile) error {
	pro, err := c.doJSONProfileReq("GET", p)
	if err != nil {
		return err
	}
	if pro != nil {
		*p = *pro
	}
	return nil
}

// CreateProfile creates a user profile, associating a public key in the process
func (c *Client) CreateProfile(p *registry.Profile, pk crypto.PrivKey) (*registry.Profile, error) {
	if c == nil {
		return nil, registry.ErrNoRegistry
	}

	// TODO (b5) - pass full profile
	pro, err := registry.ProfileFromPrivateKey(p, pk)
	if err != nil {
		return nil, err
	}

	return c.doJSONProfileReq("POST", pro)
}

// ProveProfileKey associates a public key with a profile by proving this user
// can sign messages with the private key
func (c *Client) ProveProfileKey(p *registry.Profile, pk crypto.PrivKey) (*registry.Profile, error) {
	if c == nil {
		return nil, registry.ErrNoRegistry
	}

	// TODO (b5) - pass full profile
	pro, err := registry.ProfileFromPrivateKey(p, pk)
	if err != nil {
		return nil, err
	}

	return c.doJSONProfileReq("PUT", pro)
}

// PutProfile adds a profile to the registry
func (c *Client) PutProfile(p *registry.Profile, privKey crypto.PrivKey) (*registry.Profile, error) {
	if c == nil {
		return nil, registry.ErrNoRegistry
	}

	p, err := registry.ProfileFromPrivateKey(p, privKey)
	if err != nil {
		return nil, err
	}

	return c.doJSONProfileReq("POST", p)
}

// DeleteProfile removes a profile from the registry
func (c *Client) DeleteProfile(p *registry.Profile, privKey crypto.PrivKey) error {
	if c == nil {
		return registry.ErrNoRegistry
	}

	p, err := registry.ProfileFromPrivateKey(p, privKey)
	if err != nil {
		return err
	}
	_, err = c.doJSONProfileReq("DELETE", p)
	return err
}

// doJSONProfileReq is a common wrapper for /profile endpoint requests
func (c Client) doJSONProfileReq(method string, p *registry.Profile) (*registry.Profile, error) {
	if c.cfg.Location == "" {
		return nil, ErrNoRegistry
	}

	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s/registry/profile", c.cfg.Location), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.httpClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "no such host") {
			return nil, ErrNoRegistry
		}
		return nil, err
	}

	// add response to an envelope
	env := struct {
		Data *registry.Profile
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
		if strings.Contains(env.Meta.Error, "taken") {
			return nil, registry.ErrUsernameTaken
		}
		return nil, fmt.Errorf("registry: %s", env.Meta.Error)
	}

	return env.Data, nil
}
