package regclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/qri/registry"
)

// GetProfile fills in missing fields in p with registry data
func (c Client) GetProfile(p *registry.Profile) error {
	pro, err := c.doJSONProfileReq("GET", p)
	if err != nil {
		return err
	}
	*p = *pro
	return nil
}

// PutProfile adds a profile to the registry
func (c Client) PutProfile(handle string, privKey crypto.PrivKey) error {
	p, err := registry.ProfileFromPrivateKey(handle, privKey)
	if err != nil {
		return err
	}
	_, err = c.doJSONProfileReq("POST", p)
	return err
}

// DeleteProfile removes a profile from the registry
func (c Client) DeleteProfile(handle string, privKey crypto.PrivKey) error {
	p, err := registry.ProfileFromPrivateKey(handle, privKey)
	if err != nil {
		return err
	}
	_, err = c.doJSONProfileReq("DELETE", p)
	return nil
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

	req, err := http.NewRequest(method, fmt.Sprintf("%s/profile", c.cfg.Location), bytes.NewReader(data))
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
		return nil, fmt.Errorf("error %d: %s", res.StatusCode, env.Meta.Error)
	}

	return env.Data, nil
}
