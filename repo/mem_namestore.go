package repo

import (
	"fmt"
	"github.com/ipfs/go-datastore"
)

// MemNamestore is an in-memory implementation of the Namestore interface
type MemNamestore map[string]datastore.Key

func (r MemNamestore) PutName(name string, path datastore.Key) error {
	r[name] = path
	return nil
}

func (r MemNamestore) GetPath(name string) (datastore.Key, error) {
	if r[name].String() == "" {
		return datastore.NewKey(""), ErrNotFound
	}
	return r[name], nil
}

func (r MemNamestore) GetName(path datastore.Key) (string, error) {
	for name, p := range r {
		if path.Equal(p) {
			return name, nil
		}
	}
	return "", ErrNotFound
}

func (r MemNamestore) DeleteName(name string) error {
	delete(r, name)
	return nil
}

func (r MemNamestore) Namespace(limit, offset int) ([]*DatasetRef, error) {
	if limit == -1 && len(r) <= 0 {
		return nil, fmt.Errorf("MemNamestore: nonpositive length")
	} else if limit == -1 {
		limit = len(r)
	}

	i := 0
	added := 0
	res := make([]*DatasetRef, limit)
	for name, path := range r {
		if i < offset {
			continue
		}

		if limit > 0 && added < limit {
			res[i] = &DatasetRef{
				Name: name,
				Path: path,
			}
			added++
		} else if added == limit {
			break
		}

		i++
	}
	return res, nil
}

func (r MemNamestore) NameCount() (int, error) {
	return len(r), nil
}
