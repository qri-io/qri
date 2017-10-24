package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/validate"
	"io/ioutil"
	"strings"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-ipfs/commands/files"
	"github.com/qri-io/cafs"
	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qri/repo"
)

type DatasetRequests struct {
	store cafs.Filestore
	repo  repo.Repo
}

func NewDatasetRequests(store cafs.Filestore, r repo.Repo) *DatasetRequests {
	return &DatasetRequests{
		store: store,
		repo:  r,
	}
}

func (d *DatasetRequests) List(p *ListParams, res *[]*repo.DatasetRef) error {
	// TODO - generate a sorted copy of keys, iterate through, respecting
	// limit & offset
	// ns, err := d.repo.Namespace()
	// ds, err := repo.DatasetsQuery(d.repo, query.Query{
	// 	Limit:  p.Limit,
	// 	Offset: p.Offset,
	// })
	replies, err := d.repo.Namespace(p.Limit, p.Offset)
	if err != nil {
		return fmt.Errorf("error getting namespace: %s", err.Error())
	}

	for i, ref := range replies {
		if i >= p.Limit {
			break
		}

		ds, err := dsfs.LoadDataset(d.store, ref.Path)
		if err != nil {
			// try one extra time...
			// TODO - remove this horrible hack
			ds, err = dsfs.LoadDataset(d.store, ref.Path)
			if err != nil {
				return fmt.Errorf("error loading path: %s, err: %s", ref.Path.String(), err.Error())
			}
		}
		replies[i].Dataset = ds
	}
	*res = replies
	return nil
}

type GetDatasetParams struct {
	Path datastore.Key
	Name string
	Hash string
}

func (d *DatasetRequests) Get(p *GetDatasetParams, res *dataset.Dataset) error {
	ds, err := dsfs.LoadDataset(d.store, p.Path)
	if err != nil {
		return fmt.Errorf("error loading dataset: %s", err.Error())
	}

	*res = *ds
	return nil
}

type InitDatasetParams struct {
	Data     files.File
	Metadata files.File
	Name     string
}

func (r *DatasetRequests) InitDataset(p *InitDatasetParams, res *dataset.Dataset) error {
	// TODO - split this into some sort of re-readable reader instead
	// of reading the entire file
	if p.Data == nil {
		return fmt.Errorf("data file is required")
	}

	data, err := ioutil.ReadAll(p.Data)
	if err != nil {
		return fmt.Errorf("error reading file: %s", err.Error())
	}

	st, err := detect.FromReader(p.Data.FileName(), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("error determining dataset schema: %s", err.Error())
	}

	if _, _, err := validate.Data(dsio.NewRowReader(st, bytes.NewReader(data))); err != nil {
		return fmt.Errorf("data is invalid")
	}

	datakey, err := r.store.Put(memfs.NewMemfileBytes("data."+st.Format.String(), data), true)
	if err != nil {
		return fmt.Errorf("error putting data file in store: %s", err.Error())
	}

	adr := detect.Camelize(p.Data.FileName())
	if p.Name != "" {
		adr = detect.Camelize(p.Data.FileName())
	}

	ds := &dataset.Dataset{}
	if p.Metadata != nil {
		if err := json.NewDecoder(p.Metadata).Decode(ds); err != nil {
			return fmt.Errorf("error parsing metadata json: %s", err.Error())
		}
	}

	ds.Timestamp = time.Now().In(time.UTC)
	if ds.Title == "" {
		ds.Title = adr
	}
	ds.Data = datakey
	if ds.Structure == nil {
		ds.Structure = &dataset.Structure{}
	}
	ds.Structure.Assign(st, ds.Structure)

	if err := validate.Dataset(ds); err != nil {
		return err
	}

	dskey, err := dsfs.SaveDataset(r.store, ds, true)
	if err != nil {
		return fmt.Errorf("error saving dataset: %s", err.Error())
	}

	if err = r.repo.PutDataset(dskey, ds); err != nil {
		return fmt.Errorf("error putting dataset in repo: %s", err.Error())
	}

	if err = r.repo.PutName(adr, dskey); err != nil {
		return fmt.Errorf("error adding dataset name to repo: %s", err.Error())
	}

	ds, err = r.repo.GetDataset(dskey)
	if err != nil {
		return fmt.Errorf("error reading dataset: %s", err.Error())
	}

	*res = *ds
	return nil
}

type SaveParams struct {
	Name    string
	Dataset *dataset.Dataset
}

func (r *DatasetRequests) Save(p *SaveParams, res *dataset.Dataset) error {
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

func (r *DatasetRequests) Delete(p *DeleteParams, ok *bool) error {
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

func (r *DatasetRequests) StructuredData(p *StructuredDataParams, data *StructuredData) (err error) {
	var (
		file files.File
		d    []byte
	)
	ds, err := dsfs.LoadDataset(r.store, p.Path)
	if err != nil {
		return err
	}

	if p.All {
		file, err = dsfs.LoadData(r.store, ds)
	} else {
		d, err = dsfs.LoadRows(r.store, ds, p.Limit, p.Offset)
		file = memfs.NewMemfileBytes("data", d)
	}

	if err != nil {
		return err
	}

	st := &dataset.Structure{}
	st.Assign(ds.Structure, &dataset.Structure{
		Format: p.Format,
		FormatConfig: &dataset.JsonOptions{
			ObjectEntries: p.Objects,
		},
	})

	buf := dsio.NewBuffer(st)
	rr := dsio.NewRowReader(ds.Structure, file)
	if err = dsio.EachRow(rr, func(i int, row [][]byte, err error) error {
		if err != nil {
			return err
		}
		return buf.WriteRow(row)
	}); err != nil {
		return fmt.Errorf("row iteration error: %s", err.Error())
	}

	if err := buf.Close(); err != nil {
		return fmt.Errorf("error closing row buffer: %s", err.Error())
	}

	*data = StructuredData{
		Path: p.Path,
		Data: json.RawMessage(buf.Bytes()),
	}
	return nil
}

type AddParams struct {
	Name string
	Hash string
}

func (r *DatasetRequests) AddDataset(p *AddParams, res *repo.DatasetRef) (err error) {
	fs, ok := r.store.(*ipfs.Filestore)
	if !ok {
		return fmt.Errorf("can only add datasets when running an IPFS filestore")
	}

	hash := strings.TrimSuffix(p.Hash, "/"+dsfs.PackageFileDataset.String())
	key := datastore.NewKey(hash)
	_, err = fs.Fetch(cafs.SourceAny, key)
	if err != nil {
		return fmt.Errorf("error fetching file: %s", err.Error())
	}

	err = fs.Pin(key, true)
	if err != nil {
		return fmt.Errorf("error pinning root key: %s", err.Error())
	}

	path := datastore.NewKey(key.String() + "/" + dsfs.PackageFileDataset.String())
	err = r.repo.PutName(p.Name, path)
	if err != nil {
		return fmt.Errorf("error putting dataset name in repo: %s", err.Error())
	}

	ds, err := dsfs.LoadDataset(r.store, path)
	if err != nil {
		return fmt.Errorf("error loading newly saved dataset path: %s", path.String())
	}

	*res = repo.DatasetRef{
		Name:    p.Name,
		Path:    path,
		Dataset: ds,
	}
	return
}
