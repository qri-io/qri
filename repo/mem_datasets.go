package repo

import (
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/dataset"
)

// MemDatasets is an in-memory implementation of the DatasetStore interface
type MemDatasets map[string]*dataset.Dataset

// PutDataset adds a dataset to the store, referenced by a given key
func (d MemDatasets) PutDataset(path datastore.Key, ds *dataset.Dataset) error {
	d[path.String()] = ds
	return nil
}

// PutDatasets adds multiple dataset references to the store
func (d MemDatasets) PutDatasets(datasets []*DatasetRef) error {
	for _, ds := range datasets {
		ps := ds.Path.String()
		if ps != "" {
			d[ps] = ds.Dataset
		}
	}
	return nil
}

// GetDataset fetches a dataset from the store
func (d MemDatasets) GetDataset(path datastore.Key) (*dataset.Dataset, error) {
	if d[path.String()] == nil {
		return nil, datastore.ErrNotFound
	}
	return d[path.String()], nil
}

// DeleteDataset removes a dataset from the store
func (d MemDatasets) DeleteDataset(path datastore.Key) error {
	delete(d, path.String())
	return nil
}

// Query the store for dataset results
func (d MemDatasets) Query(q query.Query) (query.Results, error) {
	re := make([]query.Entry, 0, len(d))
	for path, ds := range d {
		re = append(re, query.Entry{Key: path, Value: ds})
	}
	res := query.ResultsWithEntries(q, re)
	res = query.NaiveQueryApply(q, res)
	return res, nil
}
