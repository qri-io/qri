package repo

import (
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
)

// MemDatasets is an in-memory implementation of the DatasetStore interface
type MemDatasets struct {
	datasets map[string]*dataset.Dataset
	store    cafs.Filestore
}

// NewMemDatasets creates a datasets instance from a cafs.Filstore
func NewMemDatasets(store cafs.Filestore) MemDatasets {
	return MemDatasets{
		datasets: map[string]*dataset.Dataset{},
		store:    store,
	}
}

// PutDataset adds a dataset to the store, referenced by a given key
func (d MemDatasets) PutDataset(path datastore.Key, ds *dataset.Dataset) error {
	d.datasets[path.String()] = ds
	return nil
}

// PutDatasets adds multiple dataset references to the store
func (d MemDatasets) PutDatasets(datasets []*DatasetRef) error {
	for _, ds := range datasets {
		if ds.Path != "" {
			d.datasets[ds.Path] = ds.Dataset
		}
	}
	return nil
}

// GetDataset fetches a dataset from the store
func (d MemDatasets) GetDataset(path datastore.Key) (*dataset.Dataset, error) {
	if d.datasets[path.String()] == nil {
		return nil, datastore.ErrNotFound
	}
	return d.datasets[path.String()], nil
}

// DeleteDataset removes a dataset from the store
func (d MemDatasets) DeleteDataset(path datastore.Key) error {
	delete(d.datasets, path.String())
	return nil
}

// Query the store for dataset results
func (d MemDatasets) Query(q query.Query) (query.Results, error) {
	re := make([]query.Entry, 0, len(d.datasets))
	for path, ds := range d.datasets {
		re = append(re, query.Entry{Key: path, Value: ds})
	}
	res := query.ResultsWithEntries(q, re)
	res = query.NaiveQueryApply(q, res)
	return res, nil
}
