package regclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/qri-io/qri/registry"
)

// GetReputation gets the reputation of a profile using the ProfileID
func (c Client) GetReputation(id string) (*registry.ReputationResponse, error) {
	repReq := registry.NewReputation(id)
	res, err := c.doJSONReputationReq("GET", repReq)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// doJSONReputationReq is a common wrapper for /reputation endpoint requests
func (c Client) doJSONReputationReq(method string, rep *registry.Reputation) (*registry.ReputationResponse, error) {
	if c.cfg.Location == "" {
		return nil, ErrNoRegistry
	}

	data, err := json.Marshal(rep)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s/reputation", c.cfg.Location), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	env := struct {
		Data *registry.ReputationResponse
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
