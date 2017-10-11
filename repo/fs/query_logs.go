package fs_repo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
)

type QueryLog struct {
	basepath
	file  File
	store cafs.Filestore
}

func NewQueryLog(base string, file File, store cafs.Filestore) QueryLog {
	return QueryLog{basepath: basepath(base), file: file, store: store}
}

func (ql QueryLog) LogQuery(ref *repo.DatasetRef) error {
	log, err := ql.logs()
	if err != nil {
		return err
	}
	log = append(log, &repo.DatasetRef{Name: ref.Name, Path: ref.Path})
	return ql.saveFile(log, ql.file)
}

func (ql QueryLog) GetQueryLogs(limit, offset int) ([]*repo.DatasetRef, error) {
	logs, err := ql.logs()
	if err != nil {
		return nil, err
	}

	if offset > len(logs) {
		offset = len(logs)
	}
	stop := limit + offset
	if stop > len(logs) {
		stop = len(logs)
	}

	return logs[offset:stop], nil
}

// func (r QueryLog) PutDataset(path datastore.Key, ds *dataset.Dataset) error {
// 	d, err := r.logs()
// 	if err != nil {
// 		return err
// 	}
// 	d[path.String()] = ds
// 	return r.saveFile(d, r.file)
// }

// func (r QueryLog) PutQueryLog(logs []*repo.repo.DatasetRef) error {
// 	ds, err := r.logs()
// 	if err != nil {
// 		return err
// 	}
// 	for _, dr := range logs {
// 		ps := dr.Path.String()
// 		if ps != "" && dr.Dataset != nil {
// 			ds[ps] = dr.Dataset
// 		}
// 	}
// 	return r.saveFile(ds, r.file)
// }

// func (r QueryLog) GetDataset(path datastore.Key) (*dataset.Dataset, error) {
// 	ds, err := r.logs()
// 	if err != nil {
// 		return nil, err
// 	}
// 	ps := path.String()
// 	for p, d := range ds {
// 		if ps == p {
// 			return d, nil
// 		}
// 	}
// 	if r.store != nil {
// 		return dsfs.LoadDataset(r.store, path)
// 	}

// 	return nil, datastore.ErrNotFound
// }

// func (r QueryLog) DeleteDataset(path datastore.Key) error {
// 	ds, err := r.logs()
// 	if err != nil {
// 		return err
// 	}
// 	delete(ds, path.String())
// 	return r.saveFile(ds, r.file)
// }

// func (r QueryLog) Query(q query.Query) (query.Results, error) {
// 	ds, err := r.logs()
// 	if err != nil {
// 		return nil, err
// 	}

// 	re := make([]query.Entry, 0, len(ds))
// 	for path, d := range ds {
// 		re = append(re, query.Entry{Key: path, Value: d})
// 	}
// 	res := query.ResultsWithEntries(q, re)
// 	res = query.NaiveQueryApply(q, res)
// 	return res, nil
// }

func (r *QueryLog) logs() ([]*repo.DatasetRef, error) {
	ds := []*repo.DatasetRef{}
	data, err := ioutil.ReadFile(r.filepath(r.file))
	if err != nil {
		if os.IsNotExist(err) {
			return ds, nil
		}
		return ds, fmt.Errorf("error loading logs: %s", err.Error())
	}

	if err := json.Unmarshal(data, &ds); err != nil {
		return ds, fmt.Errorf("error unmarshaling logs: %s", err.Error())
	}
	return ds, nil
}
