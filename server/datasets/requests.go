package datasets

import (
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-ipfs/commands/files"
	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/load"
	"github.com/qri-io/dataset/writers"
	"github.com/qri-io/qri/repo"
)

func NewRequests(store cafs.Filestore, r repo.Repo) *Requests {
	return &Requests{
		store: store,
		repo:  r,
	}
}

type Requests struct {
	store cafs.Filestore
	repo  repo.Repo
}

type ListParams struct {
	OrderBy string
	Limit   int
	Offset  int
}

func (d *Requests) List(p *ListParams, res *[]*repo.DatasetRef) error {
	replies := make([]*repo.DatasetRef, p.Limit)
	i := 0
	// TODO - generate a sorted copy of keys, iterate through, respecting
	// limit & offset
	// ns, err := d.repo.Namespace()
	// ds, err := repo.DatasetsQuery(d.repo, query.Query{
	// 	Limit:  p.Limit,
	// 	Offset: p.Offset,
	// })
	names, err := d.repo.Namespace(p.Limit, p.Offset)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	for name, path := range names {
		if i >= p.Limit {
			break
		}

		ds, err := dsfs.LoadDataset(d.store, path)
		if err != nil {
			fmt.Println("error loading path:", path)
			return err
		}
		replies[i] = &repo.DatasetRef{
			Name:    name,
			Path:    path,
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
	ds, err := dsfs.LoadDataset(d.store, p.Path)
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

	path, err := dsfs.SaveDataset(r.store, ds, true)
	if err != nil {
		return err
	}

	if err := r.repo.PutName(p.Name, path); err != nil {
		return err
	}
	if err := r.repo.PutDataset(path, ds); err != nil {
		return err
	}

	*res = *ds
	return nil
}

type DeleteParams struct {
	Path datastore.Key
	Name string
}

func (r *Requests) Delete(p *DeleteParams, ok *bool) error {
	// TODO - restore
	// if p.Path.String() == "" {
	// 	r.
	// }
	// TODO - unpin resource and data
	// resource := p.Dataset.Resource
	// npath, err := r.repo.GetPath(p.Name)

	// err := r.repo.DeleteName(p.Name)
	// ns, err := r.repo.Namespace()
	// if err != nil {
	// 	return err
	// }
	// if p.Name == "" && p.Path.String() != "" {
	// 	for name, val := range ns {
	// 		if val.Equal(p.Path) {
	// 			p.Name = name
	// 		}
	// 	}
	// }

	// if p.Name == "" {
	// 	return fmt.Errorf("couldn't find dataset: %s", p.Path.String())
	// } else if ns[p.Name] == datastore.NewKey("") {
	// 	return fmt.Errorf("couldn't find dataset: %s", p.Name)
	// }

	// delete(ns, p.Name)
	// if err := r.repo.SaveNamespace(ns); err != nil {
	// 	return err
	// }
	// *ok = true
	// return nil
	return fmt.Errorf("delete dataset not yet finished")
}

type StructuredDataParams struct {
	Format        dataset.DataFormat
	Path          datastore.Key
	Objects       bool
	Limit, Offset int
	All           bool
}

type StructuredData struct {
	Path datastore.Key `json:"path"`
	Data interface{}   `json:"data"`
}

func (r *Requests) StructuredData(p *StructuredDataParams, data *StructuredData) (err error) {
	var (
		file files.File
		d    []byte
	)
	ds, err := dsfs.LoadDataset(r.store, p.Path)
	if err != nil {
		return err
	}

	if p.All {
		file, err = dsfs.LoadDatasetData(r.store, ds)
	} else {
		d, err = load.RawDataRows(r.store, ds, p.Limit, p.Offset)
		file = memfs.NewMemfileBytes("data", d)
	}

	if err != nil {
		return err
	}

	w := writers.NewJsonWriter(ds.Structure, p.Objects)
	load.EachRow(ds.Structure, file, func(i int, row [][]byte, err error) error {
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
