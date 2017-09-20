package fs_repo

import (
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/dataset"
	"io/ioutil"
	"os"
)

type DatasetStore struct {
	basepath
	file File
}

func NewDatasetStore(base string, file File) DatasetStore {
	return DatasetStore{basepath: basepath(base), file: file}
}

func (r DatasetStore) PutDataset(path string, ds *dataset.Dataset) error {
	d, err := r.datasets()
	if err != nil {
		return err
	}
	d[path] = ds
	return r.saveFile(d, r.file)
}

func (r DatasetStore) GetDataset(path string) (*dataset.Dataset, error) {
	ds, err := r.datasets()
	if err != nil {
		return nil, err
	}
	for p, d := range ds {
		if path == p {
			return d, nil
		}
	}

	return nil, datastore.ErrNotFound
}

func (r DatasetStore) DeleteDataset(path string) error {
	ds, err := r.datasets()
	if err != nil {
		return err
	}
	delete(ds, path)
	return r.saveFile(ds, r.file)
}

func (r DatasetStore) Query(q query.Query) (query.Results, error) {
	ds, err := r.datasets()
	if err != nil {
		return nil, err
	}

	re := make([]query.Entry, 0, len(ds))
	for path, d := range ds {
		re = append(re, query.Entry{Key: path, Value: d})
	}
	res := query.ResultsWithEntries(q, re)
	res = query.NaiveQueryApply(q, res)
	return res, nil
}

func (r *DatasetStore) datasets() (map[string]*dataset.Dataset, error) {
	ds := map[string]*dataset.Dataset{}
	data, err := ioutil.ReadFile(r.filepath(r.file))
	if err != nil {
		if os.IsNotExist(err) {
			return ds, nil
		}
		return ds, fmt.Errorf("error loading datasets: %s", err.Error())
	}

	if err := json.Unmarshal(data, &ds); err != nil {
		return ds, fmt.Errorf("error unmarshaling datasets: %s", err.Error())
	}
	return ds, nil
}
