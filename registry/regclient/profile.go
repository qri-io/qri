package regclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qri/registry"
)

const proveKeyAPIEndpoint = "/registry/provekey"

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

// ProveKeyForProfile proves to the registry that the user owns the profile and
// is associating a new keypair
func (c *Client) ProveKeyForProfile(p *registry.Profile) (map[string]string, error) {
	if c == nil {
		return nil, registry.ErrNoRegistry
	}

	// Ensure all required fields are set
	if p.ProfileID == "" {
		return nil, fmt.Errorf("ProveKeyForProfile: ProfileID required")
	}
	if p.Username == "" {
		return nil, fmt.Errorf("ProveKeyForProfile: Username required")
	}
	if p.Email == "" {
		return nil, fmt.Errorf("ProveKeyForProfile: Email required")
	}
	if p.Password == "" {
		return nil, fmt.Errorf("ProveKeyForProfile: Password required")
	}
	if p.PublicKey == "" {
		return nil, fmt.Errorf("ProveKeyForProfile: PublicKey required")
	}
	if p.Signature == "" {
		return nil, fmt.Errorf("ProveKeyForProfile: Signature required")
	}

	// Send proof request to the registry
	res := RegistryResponse{}
	err := c.doJSONRegistryRequest("PUT", proveKeyAPIEndpoint, p, &res)
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// UpdateProfile updates some information about the profile in the registry
func (c *Client) UpdateProfile(p *registry.Profile, pk crypto.PrivKey) (*registry.Profile, error) {
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
	// TODO(arqu): convert to lib/http/HTTPClient
	res, err := HTTPClient.Do(req)
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

	// Peername is in the process of being deprecated
	// We want to favor Username, which is what we are
	// using in all our cloud services
	// this ensures any old references to Peername will not
	// be lost
	env.Data.Peername = env.Data.Username
	return env.Data, nil
}

// RegistryResponse is a generic container for registry requests
// TODO(dustmop): Only used currently for ProveKey
// TODO(arqu): update with common API response object
type RegistryResponse struct {
	Data map[string]string
	Meta struct {
		Error  string
		Status string
		Code   int
	}
}

// doJSONProfileReq sends a json body to the registry
func (c Client) doJSONRegistryRequest(method, url string, input interface{}, output *RegistryResponse) error {
	if c.cfg.Location == "" {
		return ErrNoRegistry
	}

	data, err := json.Marshal(input)
	if err != nil {
		return err
	}

	fullurl := fmt.Sprintf("%s%s", c.cfg.Location, url)
	req, err := http.NewRequest(method, fullurl, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	// TODO(arqu): convert to lib/http/HTTPClient
	res, err := HTTPClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "no such host") {
			return ErrNoRegistry
		}
		return err
	}

	if err := json.NewDecoder(res.Body).Decode(output); err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("registry: %s", output.Meta.Error)
	}
	return nil
}
