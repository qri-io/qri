package lib

import (
	"context"
	"fmt"
	"net/http"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/registry/regclient"
	"github.com/qri-io/qri/repo"
)

// SearchMethods encapsulates business logic for the qri search command
// TODO (b5): switch to using an Instance instead of separate fields
type SearchMethods struct {
	inst *Instance
}

// NewSearchMethods creates SearchMethods from a qri Instance
func NewSearchMethods(inst *Instance) *SearchMethods {
	return &SearchMethods{inst: inst}
}

// CoreRequestsName implements the requests
func (m SearchMethods) CoreRequestsName() string { return "search" }

// SearchParams defines paremeters for the search Method
type SearchParams struct {
	QueryString string `json:"q"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *SearchParams) UnmarshalFromRequest(r *http.Request) error {
	if p == nil {
		p = &SearchParams{}
	}

	lp := &ListParams{}
	if err := lp.UnmarshalFromRequest(r); err != nil {
		return err
	}

	p.Limit = lp.Limit
	p.Offset = lp.Offset

	if p.QueryString == "" {
		p.QueryString = r.FormValue("q")
	}

	return nil
}

// SearchResult struct
type SearchResult struct {
	Type, ID string
	URL      string
	Value    *dataset.Dataset
}

// Search queries for items on qri related to given parameters
func (m *SearchMethods) Search(ctx context.Context, p *SearchParams) ([]SearchResult, error) {
	if m.inst.http != nil {
		res := []SearchResult{}
		err := m.inst.http.Call(ctx, AESearch, p, &res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}
	if p == nil {
		return nil, fmt.Errorf("error: search params cannot be nil")
	}

	reg := m.inst.registry
	if reg == nil {
		return nil, repo.ErrNoRegistry
	}
	params := &regclient.SearchParams{
		QueryString: p.QueryString,
		Limit:       p.Limit,
		Offset:      p.Offset,
	}

	regResults, err := reg.Search(params)
	if err != nil {
		return nil, err
	}

	searchResults := make([]SearchResult, len(regResults))
	for i, result := range regResults {
		searchResults[i].Type = "dataset"
		searchResults[i].ID = result.Path
		searchResults[i].Value = result
	}
	return searchResults, nil
}
