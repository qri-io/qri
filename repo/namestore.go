package repo

import (
	"github.com/ipfs/go-datastore"
)

// Namespace is an in-progress solution for aliasing
// datasets locally
type Namestore interface {
	PutName(name string, path datastore.Key) error
	GetPath(name string) (datastore.Key, error)
	GetName(path datastore.Key) (string, error)
	DeleteName(name string) error
	Names(limit, offset int) (map[string]datastore.Key, error)
	NameCount() (int, error)
}

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

func (r MemNamestore) Names(limit, offset int) (map[string]datastore.Key, error) {
	i := 0
	added := 0
	res := map[string]datastore.Key{}
	for name, path := range r {
		if i < offset {
			continue
		}

		if limit > 0 && added < limit {
			res[name] = path
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
