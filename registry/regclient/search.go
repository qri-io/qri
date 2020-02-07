package regclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/registry"
)

// SearchFilter stores various types of filters that may be applied
// to a search
type SearchFilter struct {
	// Type denotes the ype of search filter
	Type string
	// Relation indicates the relation between the key and value
	// supported options include ["eq"|"neq"|"gt"|"gte"|"lt"|"lte"]
	Relation string
	// Key corresponds to the name of the index mapping that we wish to
	// apply the filter to
	Key string
	// Value is the predicate of the subject-relation-predicate triple
	// eg. [key=timestamp] [gte] [value=[today]]
	Value interface{}
}

// SearchParams contains the parameters that are passed to a
// Client.Search method
type SearchParams struct {
	QueryString string
	Filters     []SearchFilter
	Limit       int
	Offset      int
}

// Search makes a registry search request
func (c Client) Search(p *SearchParams) ([]*dataset.Dataset, error) {
	params := &registry.SearchParams{
		Q: p.QueryString,
		//Filters: p.Filters,
		Limit:  p.Limit,
		Offset: p.Offset,
	}
	results, err := c.doJSONSearchReq("GET", params)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (c Client) prepPostReq(s *registry.SearchParams) (*http.Request, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/registry/search", c.cfg.Location), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c Client) prepGetReq(s *registry.SearchParams) (*http.Request, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/registry/search", c.cfg.Location), nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("q", s.Q)
	if s.Limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", s.Limit))
	}
	if s.Offset > -1 {
		q.Add("offset", fmt.Sprintf("%d", s.Offset))
	}
	req.URL.RawQuery = q.Encode()
	return req, nil
}

func (c Client) doJSONSearchReq(method string, s *registry.SearchParams) (results []*dataset.Dataset, err error) {
	if c.cfg.Location == "" {
		return nil, ErrNoRegistry
	}
	// req := &http.Request{}
	var req *http.Request
	switch method {
	case "POST":
		req, err = c.prepPostReq(s)
	case "GET":
		req, err = c.prepGetReq(s)
	}
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
		Data []*dataset.Dataset
		Meta struct {
			Error  string
			Status string
			Code   int
		}
	}{}

	data, _ := ioutil.ReadAll(res.Body)
	fmt.Printf("res: %s", string(data))

	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&env); err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error %d: %s", res.StatusCode, env.Meta.Error)
	}
	return env.Data, nil
}
