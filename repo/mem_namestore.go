package repo

import (
	"sort"

	"github.com/ipfs/go-datastore"
)

// MemNamestore is an in-memory implementation of the Namestore interface
type MemNamestore []*DatasetRef

func (r *MemNamestore) PutName(name string, path datastore.Key) error {
	for _, ref := range *r {
		if ref.Name == name {
			ref.Path = path
			return nil
		}
	}
	*r = append(*r, &DatasetRef{
		Name: name,
		Path: path,
	})
	sl := *r
	sort.Slice(sl, func(i, j int) bool { return sl[i].Name < sl[j].Name })
	*r = sl
	return nil
}

func (r MemNamestore) GetPath(name string) (datastore.Key, error) {
	for _, ref := range r {
		if ref.Name == name {
			return ref.Path, nil
		}
	}
	return datastore.NewKey(""), ErrNotFound
}

func (r MemNamestore) GetName(path datastore.Key) (string, error) {
	for _, ref := range r {
		if ref.Path.Equal(path) {
			return ref.Name, nil
		}
	}
	return "", ErrNotFound
}

func (r MemNamestore) DeleteName(name string) error {
	for i, ref := range r {
		if ref.Name == name {
			r = append(r[:i], r[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (r MemNamestore) Namespace(limit, offset int) ([]*DatasetRef, error) {
	res := make([]*DatasetRef, limit)
	for i, ref := range r {
		if i < offset {
			continue
		}
		if i-offset == limit {
			return res, nil
		}
		res[i-offset] = &DatasetRef{
			Name: ref.Name,
			Path: ref.Path,
		}
	}
	return res[:len(r)-offset], nil
}

func (r MemNamestore) NameCount() (int, error) {
	return len(r), nil
}
