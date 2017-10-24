package search

import (
	"strings"

	"github.com/blevesearch/bleve"
	//_ "github.com/blevesearch/bleve/config"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo"
)

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
		res[i] = &repo.DatasetRef{Path: datastore.NewKey(hit.ID)}
	}

	// fmt.Println(searchResults)
	return res, nil
}

func NewDatasetQuery(q string, limit, offset int) query.Query {
	return query.Query{
		Filters: []query.Filter{
			DatasetSearchFilter{query: q, lowered: strings.ToLower(q)},
		},
		Limit:  limit,
		Offset: offset,
	}
}

type DatasetSearchFilter struct {
	query   string
	lowered string
}

func (f DatasetSearchFilter) Filter(e query.Entry) bool {
	ds, ok := e.Value.(*dataset.Dataset)
	if !ok {
		return false
	}

	q := f.lowered
	if strings.Contains(strings.ToLower(ds.Title), q) ||
		strings.Contains(strings.ToLower(ds.Description), q) {
		return true
	}

	return false
}
