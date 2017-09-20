package datasets

import (
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/castore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/load"
	"github.com/qri-io/dataset/writers"
	"github.com/qri-io/qri/repo"
)

func NewRequests(store castore.Datastore, r repo.Repo) *Requests {
	return &Requests{
		store: store,
		repo:  r,
	}
}

type Requests struct {
	store castore.Datastore
	repo  repo.Repo
}

type ListParams struct {
	OrderBy string
	Limit   int
	Offset  int
}

func (d *Requests) List(p *ListParams, res *[]*dataset.DatasetRef) error {
	replies := make([]*dataset.DatasetRef, p.Limit)
	i := 0
	// TODO - generate a sorted copy of keys, iterate through, respecting
	// limit & offset
	// ns, err := d.repo.Namespace()
	ds, err := repo.QueryDatasets(d.repo, query.Query{
		Limit:  p.Limit,
		Offset: p.Offset,
	})
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	for name, key := range ns {
		if i >= p.Limit {
			break
		}

		// v, err := d.store.Get(key)
		// if err != nil {
		// 	return err
		// }
		// // structure, err := dataset.UnmarshalStructure(v)
		// _, err = dataset.UnmarshalStructure(v)
		// if err != nil {
		// 	return err
		// }
		ds, err := dataset.LoadDataset(d.store, key)
		if err != nil {
			fmt.Println("error loading path:", key)
			return err
		}
		replies[i] = &dataset.DatasetRef{
			Name:    name,
			Path:    key,
			Dataset: ds,
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
	ds, err := dataset.LoadDataset(d.store, p.Path)
	if err != nil {
		return err
	}

	*res = *ds
	return nil
}

type SaveParams struct {
	Name    string
	Dataset *dataset.Dataset
}

func (r *Requests) Save(p *SaveParams, res *dataset.Dataset) error {
	ds := p.Dataset

	key, err := ds.Save(r.store)
	if err != nil {
		return err
	}

	if err := r.repo.PutDataset(p.Name, ds); err != nil {
		return err
	}

	// ns, err := r.repo.Namespace()
	// if err != nil {
	// 	return err
	// }
	// ns[p.Name] = key
	// if err := r.repo.SaveNamespace(ns); err != nil {
	// 	return err
	// }

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
	Format        dataset.DataFormat
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

	w := writers.NewJsonWriter(ds.Structure, false)
	load.EachRow(ds.Structure, raw, func(i int, row [][]byte, err error) error {
		if err != nil {
			return err
		}

		if i < p.Offset {
			return nil
		} else if i-p.Offset > p.Limit {
			return fmt.Errorf("EOF")
		}

		return w.WriteRow(row)
	})

	if err := w.Close(); err != nil {
		return err
	}

	*data = StructuredData{
		Path: p.Path,
		Data: json.RawMessage(w.Bytes()),
	}
	return nil
}
