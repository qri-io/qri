package repo

import (
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
	res := make([]*DatasetRef, limit)
	i := -1
	for name, path := range r {
		i++
		if i < offset {
			continue
		}
		if i-offset == limit {
			return res, nil
		}
		res[i-offset] = &DatasetRef{
			Name: name,
			Path: path,
		}
	}
	return res[:i-offset+1], nil
}

func (r MemNamestore) NameCount() (int, error) {
	return len(r), nil
}
