package fsrepo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/search"
)

// Refstore is a file-based implementation of the repo.Refstore
// interface. It stores names in a json file
type Refstore struct {
	basepath
	// optional search index to add/remove from
	index search.Index
	// filestore for checking dataset integrity
	store cafs.Filestore
}

// PutRef adds a reference to the store
func (n Refstore) PutRef(put repo.DatasetRef) (err error) {
	var ds *dataset.Dataset

	if put.PeerID == "" {
		return repo.ErrPeerIDRequired
	} else if put.Name == "" {
		return repo.ErrNameRequired
	} else if put.Path == "" {
		return repo.ErrPathRequired
	} else if put.Peername == "" {
		return repo.ErrPeernameRequired
	}

	p := repo.DatasetRef{Peername: put.Peername, PeerID: put.PeerID, Name: put.Name, Path: put.Path}

	names, err := n.names()
	if err != nil {
		return err
	}

	for _, ref := range names {
		if ref.Equal(p) {
			return nil
		} else if ref.Match(p) {
			return repo.ErrNameTaken
		}
	}

	names = append(names, p)
	if n.store != nil {
		ds, err = dsfs.LoadDataset(n.store, datastore.NewKey(p.Path))
		if err != nil {
			return err
		}
	}

	if n.index != nil {
		batch := n.index.NewBatch()
		err = batch.Index(p.Path, ds)
		if err != nil {
			return err
		}
		err = n.index.Batch(batch)
		if err != nil {
			return err
		}
	}

	return n.save(names)
}

// GetRef completes a partially-known reference
func (n Refstore) GetRef(get repo.DatasetRef) (repo.DatasetRef, error) {
	names, err := n.names()
	if err != nil {
		return repo.DatasetRef{}, err
	}
	for _, ref := range names {
		if ref.Match(get) {
			return ref, nil
		}
	}
	return repo.DatasetRef{}, repo.ErrNotFound
}

// DeleteRef removes a name from the store
func (n Refstore) DeleteRef(del repo.DatasetRef) error {
	names, err := n.names()
	if err != nil {
		return err
	}

	for i, ref := range names {
		if ref.Match(del) {
			if ref.Path != "" && n.index != nil {
				if err := n.index.Delete(ref.Path); err != nil {
					return err
				}
			}
			names = append(names[:i], names[i+1:]...)
			break
		}
	}

	return n.save(names)
}

// References gives a set of dataset references from the store
func (n Refstore) References(limit, offset int) ([]repo.DatasetRef, error) {
	names, err := n.names()
	if err != nil {
		return nil, err
	}
	res := make([]repo.DatasetRef, limit)
	for i, ref := range names {
		if i < offset {
			continue
		}
		if i-offset == limit {
			return res, nil
		}
		res[i-offset] = ref
	}
	return res[:len(names)-offset], nil
}

// RefCount returns the size of the Refstore
func (n Refstore) RefCount() (int, error) {
	names, err := n.names()
	if err != nil {
		return 0, err
	}
	return len(names), nil
}

func (n *Refstore) names() ([]repo.DatasetRef, error) {
	data, err := ioutil.ReadFile(n.filepath(FileRefstore))
	if err != nil {
		if os.IsNotExist(err) {
			// empty is ok
			return []repo.DatasetRef{}, nil
		}
		return nil, fmt.Errorf("error loading names: %s", err.Error())
	}

	refs := []string{}
	if err := json.Unmarshal(data, &refs); err != nil {
		prevns := []repo.DatasetRef{}
		if err := json.Unmarshal(data, &prevns); err != nil {
			return nil, fmt.Errorf("error unmarshaling names: %s", err.Error())
		}
		return prevns, nil
	}

	ns := make([]repo.DatasetRef, len(refs))
	for i, rs := range refs {
		ref, err := repo.ParseDatasetRef(rs)
		if err != nil {
			// hold over for
			// TODO - remove by 0.3.0
			if err.Error() == "invalid PeerID: 'ipfs'" {
				ref.PeerID = ""
				ref.Path = fmt.Sprintf("/ipfs/%s", ref.Path)
				ns[i] = ref
				continue
			}
			return nil, err
		}
		ns[i] = ref
	}

	return ns, nil
}

func (n *Refstore) save(ns []repo.DatasetRef) error {
	strs := make([]string, len(ns))
	for i, ref := range ns {
		strs[i] = ref.String()
	}
	strs = sort.StringSlice(strs)
	return n.saveFile(strs, FileRefstore)
}
