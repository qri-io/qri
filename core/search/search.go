package search

import (
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	"strings"
)

func Search(r repo.Repo, store cafs.Filestore, q query.Query) ([]*repo.DatasetRef, error) {
	results := make([]*repo.DatasetRef, q.Limit)

	// TODO - lol 10000?
	ns, err := r.Namespace(10000, 0)
	if err != nil {
		return nil, err
	}

	i := 0
NAMES:
	for name, path := range ns {
		if i == q.Limit {
			break
		}

		ds, err := dsfs.LoadDataset(store, path)
		if err != nil {
			return nil, err
		}

		entry := query.Entry{
			Key:   path.String(),
			Value: ds,
		}

		for _, f := range q.Filters {
			if !f.Filter(entry) {
				continue NAMES
			}
		}

		results[i] = &repo.DatasetRef{
			Name:    name,
			Path:    path,
			Dataset: ds,
		}
		i++
	}

	results = results[:i]

	return results, nil
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
