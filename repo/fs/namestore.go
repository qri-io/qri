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

// Namestore is a file-based implementation of the repo.Namestore
// interface. It stores names in a json file
type Namestore struct {
	basepath
	// optional search index to add/remove from
	index search.Index
	// filestore for checking dataset integrity
	store cafs.Filestore
}

// PutName adds a name to the store
func (n Namestore) PutName(name string, path datastore.Key) (err error) {
	var ds *dataset.Dataset

	if name == "" {
		return repo.ErrNameRequired
	}

	names, err := n.names()
	if err != nil {
		return err
	}

	for _, ref := range names {
		if ref.Name == name {
			return repo.ErrNameTaken
		}
	}

	r := &repo.DatasetRef{
		Name: name,
		Path: path.String(),
	}
	names = append(names, r)

	if n.store != nil {
		ds, err = dsfs.LoadDataset(n.store, path)
		if err != nil {
			return err
		}
	}

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

	return n.save(names)
}

// GetPath gets a path for a given name
func (n Namestore) GetPath(name string) (datastore.Key, error) {
	names, err := n.names()
	if err != nil {
		return datastore.NewKey(""), err
	}
	for _, ref := range names {
		if ref.Name == name {
			return datastore.NewKey(ref.Path), nil
		}
	}
	return datastore.NewKey(""), repo.ErrNotFound
}

// GetName gets a name for a given path
func (n Namestore) GetName(path datastore.Key) (string, error) {
	names, err := n.names()
	if err != nil {
		return "", err
	}
	p := path.String()
	for _, ref := range names {
		if ref.Path == p {
			return ref.Name, nil
		}
	}
	return "", repo.ErrNotFound
}

// DeleteName removes a name from the store
func (n Namestore) DeleteName(name string) error {
	names, err := n.names()
	if err != nil {
		return err
	}

	for i, ref := range names {
		if ref.Name == name {
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

// Namespace gives a set of dataset references from the store
func (n Namestore) Namespace(limit, offset int) ([]*repo.DatasetRef, error) {
	names, err := n.names()
	if err != nil {
		return nil, err
	}
	res := make([]*repo.DatasetRef, limit)
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

// NameCount returns the size of the Namestore
func (n Namestore) NameCount() (int, error) {
	names, err := n.names()
	if err != nil {
		return 0, err
	}
	return len(names), nil
}

func (n *Namestore) names() ([]*repo.DatasetRef, error) {
	ns := []*repo.DatasetRef{}
	data, err := ioutil.ReadFile(n.filepath(FileNamestore))
	if err != nil {
		if os.IsNotExist(err) {
			return ns, nil
		}
		return ns, fmt.Errorf("error loading names: %s", err.Error())
	}

	if err := json.Unmarshal(data, &ns); err != nil {
		prevns := map[string]datastore.Key{}
		if err := json.Unmarshal(data, &prevns); err != nil {
			return ns, fmt.Errorf("error unmarshaling names: %s", err.Error())
		}
		ns = make([]*repo.DatasetRef, len(prevns))
		i := 0
		for name, path := range prevns {
			ns[i] = &repo.DatasetRef{
				Name: name,
				Path: path.String(),
			}
			i++
		}
		sort.Slice(ns, func(i, j int) bool { return ns[i].Name < ns[j].Name })
	}

	return ns, nil
}

func (n *Namestore) save(ns []*repo.DatasetRef) error {
	sort.Slice(ns, func(i, j int) bool { return ns[i].Name < ns[j].Name })
	return n.saveFile(ns, FileNamestore)
}
