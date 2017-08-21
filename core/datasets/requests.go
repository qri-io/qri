package datasets

import (
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/castore/ipfs"
	"github.com/qri-io/dataset/load"
	"github.com/qri-io/qri/repo"
	// "github.com/qri-io/castore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/core"
)

func NewRequests(store *ipfs_datastore.Datastore, r repo.Repo) *Requests {
	return &Requests{
		store: store,
		repo:  r,
	}
}

type Requests struct {
	store *ipfs_datastore.Datastore
	repo  repo.Repo
}

type ListParams struct {
	OrderBy string
	Limit   int
	Offset  int
}

func (d *Requests) List(p *ListParams, res *[]*dataset.Dataset) error {
	replies := make([]*dataset.Dataset, p.Limit)
	i := 0
	// TODO - generate a sorted copy of keys, iterate through, respecting
	// limit & offset
	ns, err := d.repo.Namespace()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	for name, key := range ns {
		if i >= p.Limit {
			break
		}

		v, err := d.store.Get(key)
		if err != nil {
			return err
		}
		// structure, err := dataset.UnmarshalStructure(v)
		_, err = dataset.UnmarshalStructure(v)
		if err != nil {
			return err
		}
		replies[i] = &dataset.Dataset{
			Title: name,
			// TODO - need to figure out how to deref a dataset
			// Path:  key,
			// into the right position
			// Structure: *resource,
		}
		i++
	}
	*res = replies[:i]
	return nil
}

type GetParams struct {
	Path datastore.Key
	Name string
	Hash string
}

func (d *Requests) Get(p *GetParams, res *dataset.Dataset) error {
	// resource, err := core.GetStructure(d.store, p.Path)
	_, err := core.GetStructure(d.store, p.Path)
	if err != nil {
		return err
	}

	*res = dataset.Dataset{
	// TODO - put back
	// Resource: *resource,
	}
	return nil
}

type SaveParams struct {
	Name    string
	Dataset *dataset.Dataset
}

func (r *Requests) Save(p *SaveParams, res *dataset.Dataset) error {
	ds := p.Dataset

	dsdata, err := ds.MarshalJSON()
	if err != nil {
		return err
	}
	dshash, err := r.store.AddAndPinBytes(dsdata)
	if err != nil {
		return err
	}

	ns, err := r.repo.Namespace()
	if err != nil {
		return err
	}
	ns[p.Name] = datastore.NewKey("/ipfs/" + dshash)
	if err := r.repo.SaveNamespace(ns); err != nil {
		return err
	}

	*res = *ds
	return nil
}

type DeleteParams struct {
	Name string
	Path datastore.Key
}

func (r *Requests) Delete(p *DeleteParams, ok *bool) error {
	// TODO - unpin resource and data
	// resource := p.Dataset.Resource
	ns, err := r.repo.Namespace()
	if err != nil {
		return err
	}
	if p.Name == "" && p.Path.String() != "" {
		for name, val := range ns {
			if val.Equal(p.Path) {
				p.Name = name
			}
		}
	}

	if p.Name == "" {
		return fmt.Errorf("couldn't find dataset: %s", p.Path.String())
	} else if ns[p.Name] == datastore.NewKey("") {
		return fmt.Errorf("couldn't find dataset: %s", p.Name)
	}

	delete(ns, p.Name)
	if err := r.repo.SaveNamespace(ns); err != nil {
		return err
	}
	*ok = true
	return nil
}

type StructuredDataParams struct {
	Path          datastore.Key
	Limit, Offset int
	All           bool
}

type StructuredData struct {
	Path datastore.Key `json:"path"`
	Data interface{}   `json:"data"`
}

func (r *Requests) StructuredData(p *StructuredDataParams, data *StructuredData) (err error) {
	var raw []byte
	ds, err := dataset.LoadDataset(r.store, p.Path)
	if err != nil {
		return err
	}

	if p.All {
		raw, err = ds.LoadData(r.store)
	} else {
		raw, err = load.RawDataRows(r.store, ds, p.Limit, p.Offset)
	}

	if err != nil {
		return err
	}

	*data = StructuredData{
		Path: p.Path,
		Data: string(raw),
	}
	return nil
}
