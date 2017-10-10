package search

import (
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/repo"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/dataset"
)

func Search(i Index, q string) ([]*repo.DatasetRef, error) {

	query := bleve.NewQueryStringQuery(q)
	search := bleve.NewSearchRequest(query)
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

	// 	// TODO - lol 10000?
	// 	ns, err := r.Namespace(10000, 0)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	i := 0
	// NAMES:
	// 	for name, path := range ns {
	// 		if i == q.Limit {
	// 			break
	// 		}

	// 		ds, err := dsfs.LoadDataset(store, path)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		entry := query.Entry{
	// 			Key:   path.String(),
	// 			Value: ds,
	// 		}

	// 		for _, f := range q.Filters {
	// 			if !f.Filter(entry) {
	// 				continue NAMES
	// 			}
	// 		}

	// 		results[i] = &repo.DatasetRef{
	// 			Name:    name,
	// 			Path:    path,
	// 			Dataset: ds,
	// 		}
	// 		i++
	// 	}
	// 	results = results[:i]

	// return results, nil
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
