package regclient

import (
	"context"

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
	Query   string
	Filters []SearchFilter
	Limit   int
	Offset  int
}

// Search makes a registry search request
func (c Client) Search(ctx context.Context, p *SearchParams) ([]*dataset.Dataset, error) {
	if c.httpClient == nil {
		return nil, ErrNoRegistry
	}

	params := &registry.SearchParams{
		Q: p.Query,
		//Filters: p.Filters,
		Limit:  p.Limit,
		Offset: p.Offset,
	}
	results := []*dataset.Dataset{}
	err := c.httpClient.CallMethod(ctx, "registry/search", "GET", "", params, results)
	if err != nil {
		return nil, err
	}
	return results, nil
}
