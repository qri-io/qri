// Package search creates a local search index of a repo using the bleve search engine
package search

import (
	"github.com/qri-io/bleve"
	"github.com/qri-io/qri/repo"
)

// Search searches this repo's bleve index
func Search(i Index, p repo.SearchParams) ([]repo.DatasetRef, error) {
	query := bleve.NewQueryStringQuery(p.Q)
	search := bleve.NewSearchRequest(query)
	//TODO: find better place to set default, and/or expose option
	search.Size = p.Limit
	search.From = p.Offset
	results, err := i.Search(search)
	if err != nil {
		return nil, err
	}

	res := make([]repo.DatasetRef, results.Hits.Len())
	for i, hit := range results.Hits {
		res[i] = repo.DatasetRef{Path: hit.ID}
	}

	return res, nil
}
