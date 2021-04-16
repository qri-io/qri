package lib

import (
	"context"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/registry/regclient"
	"github.com/qri-io/qri/repo"
)

// SearchMethods groups together methods for search
type SearchMethods struct {
	d dispatcher
}

// Name returns the name of this method group
func (m SearchMethods) Name() string {
	return "search"
}

// Attributes defines attributes for each method
func (m SearchMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"search": {AESearch, "POST", ""},
	}
}

// SearchParams defines paremeters for the search Method
type SearchParams struct {
	Query  string `json:"q"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// SearchResult struct
type SearchResult struct {
	Type, ID string
	URL      string
	Value    *dataset.Dataset
}

// Search queries for items on qri related to given parameters
func (m SearchMethods) Search(ctx context.Context, p *SearchParams) ([]SearchResult, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "search"), p)
	if res, ok := got.([]SearchResult); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Implementations for FSI methods follow

// searchImpl holds the method implementations for search
type searchImpl struct{}

// Search queries for items on qri related to given parameters
func (searchImpl) Search(scope scope, p *SearchParams) ([]SearchResult, error) {
	client := scope.RegistryClient()
	if client == nil {
		return nil, repo.ErrNoRegistry
	}
	params := &regclient.SearchParams{
		Query:  p.Query,
		Limit:  p.Limit,
		Offset: p.Offset,
	}

	regResults, err := client.Search(params)
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
