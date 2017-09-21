package repo

import (
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/dataset"
)

type MemDatasets map[string]*dataset.Dataset

func (d MemDatasets) PutDataset(path datastore.Key, ds *dataset.Dataset) error {
	d[path.String()] = ds
	return nil
}

func (d MemDatasets) PutDatasets(datasets []*dataset.DatasetRef) error {
	for _, ds := range datasets {
		ps := ds.Path.String()
		if ps != "" {
			d[ps] = ds.Dataset
		}
	}
	return nil
}

func (d MemDatasets) GetDataset(path datastore.Key) (*dataset.Dataset, error) {
	if d[path.String()] == nil {
		return nil, datastore.ErrNotFound
	}
	return d[path.String()], nil
}

func (d MemDatasets) DeleteDataset(path datastore.Key) error {
	delete(d, path.String())
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
