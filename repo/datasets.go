package repo

import (
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/dataset"
)

type MemDatasets map[string]*dataset.Dataset

func (d MemDatasets) PutDataset(path string, ds *dataset.Dataset) error {
	d[path] = ds
	return nil
}

func (d MemDatasets) GetDataset(path string) (*dataset.Dataset, error) {
	if d[path] == nil {
		return nil, datastore.ErrNotFound
	}
	return d[path], nil
}

func (d MemDatasets) DeleteDataset(path string) error {
	delete(d, path)
	return nil
}

func (d MemDatasets) Query(q query.Query) (query.Results, error) {
	re := make([]query.Entry, 0, len(d))
	for path, ds := range d {
		re = append(re, query.Entry{Key: path, Value: ds})
	}
	res := query.ResultsWithEntries(q, re)
	res = query.NaiveQueryApply(q, res)
	return res, nil
}
