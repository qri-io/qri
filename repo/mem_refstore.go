package repo

import (
	"sort"

	reporef "github.com/qri-io/qri/repo/ref"
)

// MemRefstore is an in-memory implementation of the Namestore interface
type MemRefstore []reporef.DatasetRef

// PutRef adds a reference to the namestore. Only complete references may be added
func (r *MemRefstore) PutRef(put reporef.DatasetRef) error {
	if put.ProfileID == "" {
		return ErrPeerIDRequired
	} else if put.Peername == "" {
		return ErrPeernameRequired
	} else if put.Name == "" {
		return ErrNameRequired
	} else if put.Path == "" && put.FSIPath == "" {
		return ErrPathRequired
	}

	for i, ref := range *r {
		if ref.Match(put) {
			rs := *r
			rs[i] = put
			return nil
		}
	}
	*r = append(*r, put)
	sl := *r
	sort.Slice(sl, func(i, j int) bool { return sl[i].Peername+sl[i].Name < sl[j].Peername+sl[j].Name })
	*r = sl
	return nil
}

// GetRef completes a reference with , refs can have either
// Path or Peername & Name specified, GetRef should fill out the missing pieces
func (r MemRefstore) GetRef(get reporef.DatasetRef) (ref reporef.DatasetRef, err error) {
	for _, ref := range r {
		if ref.Match(get) {
			return ref, nil
		}
	}
	err = ErrNotFound
	return
}

// DeleteRef removes a name from the store
func (r *MemRefstore) DeleteRef(del reporef.DatasetRef) error {
	refs := *r
	for i, ref := range refs {
		if ref.Match(del) {
			*r = append(refs[:i], refs[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

// References grabs a set of names from the Store's namespace
func (r MemRefstore) References(offset, limit int) ([]reporef.DatasetRef, error) {
	res := make([]reporef.DatasetRef, limit)
	for i, ref := range r {
		if i < offset {
			continue
		}
		if i-offset == limit {
			return res, nil
		}
		res[i-offset] = ref
	}
	return res[:len(r)-offset], nil
}

// RefCount returns the total number of names in the store
func (r MemRefstore) RefCount() (int, error) {
	return len(r), nil
}
