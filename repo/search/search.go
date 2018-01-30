package search

import (
	"strings"

	"github.com/qri-io/bleve"
	//_ "github.com/qri-io/bleve/config"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo"
)

// Search searches this repo's bleve index
func Search(i Index, p repo.SearchParams) ([]*repo.DatasetRef, error) {
	query := bleve.NewQueryStringQuery(p.Q)
	search := bleve.NewSearchRequest(query)
	//TODO: find better place to set default, and/or expose option
	search.Size = p.Limit
	search.From = p.Offset
	results, err := i.Search(search)
	if err != nil {
		return nil, err
	}

	res := make([]*repo.DatasetRef, results.Hits.Len())
	for i, hit := range results.Hits {
		res[i] = &repo.DatasetRef{Path: hit.ID}
	}

	// fmt.Println(searchResults)
	return res, nil
}

// NewDatasetQuery generates a query for datastes from a query string, limit, and offset
func NewDatasetQuery(q string, limit, offset int) query.Query {
	return query.Query{
		Filters: []query.Filter{
			DatasetSearchFilter{query: q, lowered: strings.ToLower(q)},
		},
		Limit:  limit,
		Offset: offset,
	}
}

// DatasetSearchFilter conforms to the bleve Filter interface
type DatasetSearchFilter struct {
	query   string
	lowered string
}

// Filter filters entries that don't match
func (f DatasetSearchFilter) Filter(e query.Entry) bool {
	ds, ok := e.Value.(*dataset.Dataset)
	if !ok {
		return false
	}

	q := f.lowered
	if strings.Contains(strings.ToLower(ds.Meta.Title), q) ||
		strings.Contains(strings.ToLower(ds.Meta.Description), q) {
		return true
	}

	return false
}
