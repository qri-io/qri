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
	file File
	// optional search index to add/remove from
	index search.Index
	// filestore for checking dataset integrity
	store cafs.Filestore
}

// PutRef adds a reference to the store
func (n Refstore) PutRef(p repo.DatasetRef) (err error) {
	var ds *dataset.Dataset

	if p.ProfileID == "" {
		return repo.ErrPeerIDRequired
	} else if p.Name == "" {
		return repo.ErrNameRequired
	} else if p.Path == "" {
		return repo.ErrPathRequired
	} else if p.Peername == "" {
		return repo.ErrPeernameRequired
	}

	names, err := n.names()
	if err != nil {
		return err
	}

	matched := false
	for i, ref := range names {
		if ref.Match(p) {
			matched = true
			names[i] = p
		}
	}

	if !matched {
		names = append(names, p)
	}

	if n.store != nil {
		ds, err = dsfs.LoadDataset(n.store, datastore.NewKey(p.Path))
		if err != nil {
			return err
		}
	}

	// TODO - move this up into base package
	if n.index != nil {
		batch := n.index.NewBatch()
		err = batch.Index(p.Path, ds)
		if err != nil {
			log.Debug(err.Error())
			return err
		}
		err = n.index.Batch(batch)
		if err != nil {
			log.Debug(err.Error())
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
					log.Debug(err.Error())
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
		log.Debug(err.Error())
		return 0, err
	}
	return len(names), nil
}

func (n *Refstore) names() ([]repo.DatasetRef, error) {
	data, err := ioutil.ReadFile(n.filepath(n.file))
	if err != nil {
		if os.IsNotExist(err) {
			// empty is ok
			return []repo.DatasetRef{}, nil
		}
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading names: %s", err.Error())
	}

	refs := []repo.DatasetRef{}
	if err := json.Unmarshal(data, &refs); err != nil {

		prevns := []string{}
		if err := json.Unmarshal(data, &prevns); err != nil {
			log.Debug(err.Error())
			return nil, fmt.Errorf("error unmarshaling names: %s", err.Error())
		}

		ns := make([]repo.DatasetRef, len(refs))
		for i, rs := range prevns {
			ref, err := repo.ParseDatasetRef(rs)
			if err != nil {
				return nil, err
			}
			ns[i] = ref
		}

		return ns, nil
	}

	return refs, nil
}

func (n *Refstore) save(ns []repo.DatasetRef) error {
	rs := refs(ns)
	sort.Sort(rs)
	return n.saveFile(rs, FileRefstore)
}

// refs is a slice of dataset refs that implements the sort interface
type refs []repo.DatasetRef

// Len returns the length of refs
func (r refs) Len() int { return len(r) }

// Less returns true if i comes before j
func (r refs) Less(i, j int) bool { return r[i].AliasString() < r[j].AliasString() }

// Swap flips the positions of i and j
func (r refs) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
