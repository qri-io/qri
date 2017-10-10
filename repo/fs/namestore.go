package fs_repo

import (
	"encoding/json"
	"fmt"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"io/ioutil"
	"os"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/search"
)

type Namestore struct {
	basepath
	// optional search index to add/remove from
	index search.Index
	// filestore for checking dataset integrity
	store cafs.Filestore
}

func NewNamestore(base string) Datasets {
	return Datasets{basepath: basepath(base)}
}

func (n Namestore) PutName(name string, path datastore.Key) (err error) {
	var ds *dataset.Dataset

	names, err := n.names()
	if err != nil {
		return err
	}

	if n.store != nil {
		ds, err = dsfs.LoadDataset(n.store, path)
		if err != nil {
			return err
		}
	}

	names[name] = path
	if n.index != nil {
		batch := n.index.NewBatch()
		err = batch.Index(path.String(), ds)
		if err != nil {
			return err
		}
		err = n.index.Batch(batch)
		if err != nil {
			return err
		}
	}

	return n.saveFile(names, FileNamestore)
}

func (n Namestore) GetPath(name string) (datastore.Key, error) {
	names, err := n.names()
	if err != nil {
		return datastore.NewKey(""), err
	}
	if names[name].String() == "" {
		return datastore.NewKey(""), repo.ErrNotFound
	}
	return names[name], nil
}

func (n Namestore) GetName(path datastore.Key) (string, error) {
	names, err := n.names()
	if err != nil {
		return "", err
	}
	for name, p := range names {
		if path.Equal(p) {
			return name, nil
		}
	}
	return "", repo.ErrNotFound
}

func (n Namestore) DeleteName(name string) error {
	names, err := n.names()
	if err != nil {
		return err
	}
	path := names[name]

	if path.String() != "" && n.index != nil {
		if err := n.index.Delete(path.String()); err != nil {
			return err
		}
	}

	delete(names, name)
	return n.saveFile(names, FileNamestore)
}

func (n Namestore) Namespace(limit, offset int) ([]*repo.DatasetRef, error) {
	names, err := n.names()
	if err != nil {
		return nil, err
	}
	if limit < 0 {
		limit = len(names)
	}

	i := 0
	added := 0
	res := make([]*repo.DatasetRef, limit)
	for name, path := range names {
		if i < offset {
			continue
		}

		if limit != 0 && added < limit {
			res[i] = &repo.DatasetRef{
				Name: name,
				Path: path,
			}
			added++
		} else if added == limit {
			break
		}

		i++
	}
	res = res[:i]
	return res, nil
}

func (n Namestore) NameCount() (int, error) {
	names, err := n.names()
	if err != nil {
		return 0, err
	}
	return len(names), nil
}

func (r *Namestore) names() (map[string]datastore.Key, error) {
	ns := map[string]datastore.Key{}
	data, err := ioutil.ReadFile(r.filepath(FileNamestore))
	if err != nil {
		if os.IsNotExist(err) {
			return ns, nil
		}
		return ns, fmt.Errorf("error loading names: %s", err.Error())
	}

	if err := json.Unmarshal(data, &ns); err != nil {
		return ns, fmt.Errorf("error unmarshaling names: %s", err.Error())
	}
	return ns, nil
}
