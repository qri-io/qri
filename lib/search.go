package lib

import (
	"fmt"

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

// SearchResult struct
type SearchResult struct {
	Type, ID string
	Value    interface{}
}

// Search queries for items on qri related to given parameters
func (m *SearchMethods) Search(p *SearchParams, results *[]SearchResult) error {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("SearchMethods.Search", p, results)
	}
	if p == nil {
		return fmt.Errorf("error: search params cannot be nil")
	}

	reg := m.inst.registry
	if reg == nil {
		return repo.ErrNoRegistry
	}
	params := &regclient.SearchParams{
		QueryString: p.QueryString,
		Limit:       p.Limit,
		Offset:      p.Offset,
	}

	regResults, err := reg.Search(params)
	if err != nil {
		return err
	}

	searchResults := make([]SearchResult, len(regResults))
	for i, result := range regResults {
		searchResults[i].Type = result.Type
		searchResults[i].ID = result.ID
		searchResults[i].Value = result.Value
	}
	*results = searchResults
	return nil
}
