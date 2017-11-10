package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/validate"
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
	// ensure valid limit value
	if p.Limit <= 0 {
		p.Limit = 25
	}
	// ensure valid offset value
	if p.Offset < 0 {
		p.Offset = 0
	}
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

func (d *DatasetRequests) Get(p *GetDatasetParams, res *repo.DatasetRef) error {
	ds, err := dsfs.LoadDataset(d.store, p.Path)
	if err != nil {
		return fmt.Errorf("error loading dataset: %s", err.Error())
	}

	name := p.Name
	if p.Path.String() != "" {
		name, _ = d.repo.GetName(p.Path)
	}

	*res = repo.DatasetRef{
		Name:    name,
		Path:    p.Path,
		Dataset: ds,
	}
	return nil
}

type InitDatasetParams struct {
	Name         string
	Url          string
	DataFilename string
	Data         io.Reader
	Metadata     io.Reader
}

func (r *DatasetRequests) InitDataset(p *InitDatasetParams, res *repo.DatasetRef) error {
	var rdr io.Reader
	var filename = p.DataFilename
	if p.Url != "" {
		res, err := http.Get(p.Url)
		if err != nil {
			return fmt.Errorf("error fetching url: %s", err.Error())
		}
		filename = filepath.Base(p.Url)
		defer res.Body.Close()
		rdr = res.Body
	} else if p.Data != nil {
		rdr = p.Data
	} else {
		return fmt.Errorf("either a file or a url is required to create a dataset")
	}

	// TODO - split this into some sort of re-readable reader instead
	// of reading the entire file
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		return fmt.Errorf("error reading file: %s", err.Error())
	}

	st, err := detect.FromReader(filename, bytes.NewReader(data))
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

	name := p.Name
	if name == "" && filename != "" {
		name = detect.Camelize(filename)
	}

	ds := &dataset.Dataset{}
	if p.Metadata != nil {
		if err := json.NewDecoder(p.Metadata).Decode(ds); err != nil {
			return fmt.Errorf("error parsing metadata json: %s", err.Error())
		}
	}

	ds.Timestamp = time.Now().In(time.UTC)
	if ds.Title == "" {
		ds.Title = name
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

	if err = r.repo.PutName(name, dskey); err != nil {
		return fmt.Errorf("error adding dataset name to repo: %s", err.Error())
	}

	ds, err = r.repo.GetDataset(dskey)
	if err != nil {
		return fmt.Errorf("error reading dataset: %s", err.Error())
	}

	*res = repo.DatasetRef{
		Name:    p.Name,
		Path:    dskey,
		Dataset: ds,
	}
	return nil
}

type SaveParams struct {
	Name    string
	Dataset *dataset.Dataset
}

// TODO - naming of "save" is ambiguous
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

func (r *DatasetRequests) Delete(p *DeleteParams, ok *bool) (err error) {
	if p.Name == "" && p.Path.String() == "" {
		return fmt.Errorf("either name or path is required")
	}

	if p.Path.String() == "" {
		p.Path, err = r.repo.GetPath(p.Name)
		if err != nil {
			return
		}
	}

	p.Name, err = r.repo.GetName(p.Path)
	if err != nil {
		return
	}

	if pinner, ok := r.store.(cafs.Pinner); ok {
		if err = pinner.Unpin(p.Path, true); err != nil {
			return
		}
	}

	if err = r.repo.DeleteName(p.Name); err != nil {
		return
	}

	*ok = true
	return nil
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
		file cafs.File
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

type ValidateDatasetParams struct {
	Name         string
	Url          string
	Path         datastore.Key
	DataFilename string
	Data         io.Reader
	Metadata     io.Reader
}

func (r *DatasetRequests) Validate(p *ValidateDatasetParams, errors *dataset.Dataset) (err error) {
	// store := Store(cmd, args)
	// errs, err := history.Validate(store)
	// ExitIfErr(err)

	// if cmd.Flag("check-links").Value.String() == "true" {
	// 	validation, data, count, err := ds.ValidateDeadLinks(Cache())
	// 	ExitIfErr(err)
	// 	if count > 0 {
	// 		PrintResults(validation, data, dataset.CsvDataFormat)
	// 	} else {
	// 		PrintSuccess("✔ All good!")
	// 	}
	// }

	// if p.Data != nil {
	// 	errr, count, err := validate.Data(r)
	// }

	// validation, data, count, err := ds.ValidateData(Cache())
	// ExitIfErr(err)
	// if count > 0 {
	// 	PrintResults(validation, data, dataset.CsvDataFormat)
	// } else {
	// 	PrintSuccess("✔ All good!")
	// }
	return fmt.Errorf("not finished")
}
