package repo

import (
	"sort"

	"github.com/ipfs/go-datastore"
)

// MemNamestore is an in-memory implementation of the Namestore interface
type MemNamestore []*DatasetRef

// PutName adds a name to the namestore
func (r *MemNamestore) PutName(name string, path datastore.Key) error {
	for _, ref := range *r {
		if ref.Name == name {
			ref.Path = path.String()
			return nil
		}
	}
	*r = append(*r, &DatasetRef{
		Name: name,
		Path: path.String(),
	})
	sl := *r
	sort.Slice(sl, func(i, j int) bool { return sl[i].Name < sl[j].Name })
	*r = sl
	return nil
}

// GetPath returns the path associated with a given name
func (r MemNamestore) GetPath(name string) (datastore.Key, error) {
	for _, ref := range r {
		if ref.Name == name {
			return datastore.NewKey(ref.Path), nil
		}
	}
	return datastore.NewKey(""), ErrNotFound
}

// GetName returns the name for a given path in the store
func (r MemNamestore) GetName(path datastore.Key) (string, error) {
	p := path.String()
	for _, ref := range r {
		if ref.Path == p {
			return ref.Name, nil
		}
	}
	return "", ErrNotFound
}

// DeleteName removes a name from the store
func (r MemNamestore) DeleteName(name string) error {
	for i, ref := range r {
		if ref.Name == name {
			r = append(r[:i], r[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

// Namespace grabs a set of names from the Store's namespace
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

// NameCount returns the total number of names in the store
func (r MemNamestore) NameCount() (int, error) {
	return len(r), nil
}
