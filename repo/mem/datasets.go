package mem_repo

import (
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset"
)

type Datasets map[string]*dataset.Dataset

func (d Datasets) PutDataset(path string, ds *dataset.Dataset) error {
	d[path] = ds
	return nil
}

func (d Datasets) GetDataset(path string) (*dataset.Dataset, error) {
	if d[path] == nil {
		return nil, datastore.ErrNotFound
	}
	return d[path], nil
}

func (d Datasets) DeleteDataset(path string) error {
	delete(d, path)
	return nil
}

func (d Datasets) Query(query.Query) (query.Results, error) {
	re := make([]query.Entry, 0, len(d))
	for path, ds := range d {
		re = append(re, query.Entry{Key: path, Value: ds})
	}
	res := query.ResultsWithEntries(q, re)
	res = query.NaiveQueryApply(q, res)
	return res, nil
}
