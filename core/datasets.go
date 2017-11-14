package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sort"
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
	repo repo.Repo
}

func NewDatasetRequests(r repo.Repo) *DatasetRequests {
	return &DatasetRequests{
		repo: r,
	}
}

func (d *DatasetRequests) List(p *ListParams, res *[]*repo.DatasetRef) error {
	store := d.repo.Store()
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

		ds, err := dsfs.LoadDataset(store, ref.Path)
		if err != nil {
			// try one extra time...
			// TODO - remove this horrible hack
			ds, err = dsfs.LoadDataset(store, ref.Path)
			if err != nil {
				return fmt.Errorf("error loading path: %s, err: %s", ref.Path.String(), err.Error())
			}
		}
		replies[i].Dataset = ds
	}

	// TODO - horrible hack to make sure results are default-sorted
	sort.Slice(replies, func(i, j int) bool { return replies[i].Dataset.Timestamp.After(replies[j].Dataset.Timestamp) })

	*res = replies
	return nil
}

type GetDatasetParams struct {
	Path datastore.Key
	Name string
	Hash string
}

func (d *DatasetRequests) Get(p *GetDatasetParams, res *repo.DatasetRef) error {
	store := d.repo.Store()
	ds, err := dsfs.LoadDataset(store, p.Path)
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

// InitDatasetParams encapsulates arguments to InitDataset
type InitDatasetParams struct {
	Name             string    // variable name for referring to this dataset. required.
	Url              string    // url to download data from. either Url or Data is required
	DataFilename     string    // filename of data file. extension is used for filetype detection
	Data             io.Reader // reader of structured data. either Url or Data is required
	MetadataFilename string    // filename of metadata file. optional.
	Metadata         io.Reader // reader of json-formatted metadata
	// TODO - add support for adding via path/hash
	// DataPath         datastore.Key // path to structured data
}

// InitDataset creates a new qri dataset from a source of data
func (r *DatasetRequests) InitDataset(p *InitDatasetParams, res *repo.DatasetRef) error {
	var (
		rdr   io.Reader
		store = r.repo.Store()
	)
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

	// TODO - need a better strategy for huge files
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

	datakey, err := store.Put(memfs.NewMemfileBytes("data."+st.Format.String(), data), true)
	if err != nil {
		return fmt.Errorf("error putting data file in store: %s", err.Error())
	}

	name := p.Name
	if name == "" && filename != "" {
		name = detect.Camelize(filename)
	}

	ds := &dataset.Dataset{}
	if p.Url != "" {
		ds.DownloadUrl = p.Url
		// if we're adding from a dataset url, set a default accrual periodicity of once a week
		// this'll set us up to re-check urls over time
		// TODO - make this configurable via a param
		ds.AccrualPeriodicity = "R/P1W"
	}
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

	dskey, err := dsfs.SaveDataset(store, ds, true)
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

type UpdateParams struct {
	Changes      *dataset.Dataset // all dataset changes. required.
	DataFilename string           // filename for new data. optional.
	Data         io.Reader        // stream of complete dataset update. optional.
}

// Update adds a history entry, updating a dataset
func (r *DatasetRequests) Update(p *UpdateParams, res *repo.DatasetRef) (err error) {
	var (
		name     string
		prevpath datastore.Key
	)
	store := r.repo.Store()
	ds := &dataset.Dataset{}

	rt, ref := dsfs.RefType(p.Changes.Previous.String())
	// allows using dataset names as "previous" fields
	if rt == "name" {
		name = ref
		prevpath, err = r.repo.GetPath(strings.Trim(ref, "/"))
		if err != nil {
			return fmt.Errorf("error getting previous dataset path: %s", err.Error())
		}
	} else {
		prevpath = datastore.NewKey(ref)
		// attempt to grab name for later if path is provided
		name, _ = r.repo.GetName(prevpath)
	}

	// read previous changes
	prev, err := r.repo.GetDataset(prevpath)
	if err != nil {
		return fmt.Errorf("error getting previous dataset: %s", err.Error())
	}

	// add all previous fields and any changes
	ds.Assign(prev, p.Changes)

	// store file if one is provided
	if p.Data != nil {
		data, err := ioutil.ReadAll(p.Data)
		if err != nil {
			return fmt.Errorf("error reading data: %s", err.Error())
		}

		path, err := store.Put(memfs.NewMemfileReader(p.DataFilename, p.Data), false)
		if err != nil {
			return fmt.Errorf("error putting data in store: %s", err.Error())
		}

		ds.Data = path
		ds.Length = len(data)
	}

	if strings.HasSuffix(prevpath.String(), dsfs.PackageFileDataset.String()) {
		ds.Previous = datastore.NewKey(strings.TrimSuffix(prevpath.String(), "/"+dsfs.PackageFileDataset.String()))
	} else {
		ds.Previous = prevpath
	}

	if err := validate.Dataset(ds); err != nil {
		return err
	}

	// TODO - should this go into the save method?
	ds.Timestamp = time.Now().In(time.UTC)
	dspath, err := dsfs.SaveDataset(store, ds, true)
	if err != nil {
		return fmt.Errorf("error saving dataset: %s", err.Error())
	}

	if name != "" {
		if err := r.repo.DeleteName(name); err != nil {
			return err
		}
		if err := r.repo.PutName(name, dspath); err != nil {
			return err
		}
	}

	*res = repo.DatasetRef{
		Name:    name,
		Path:    dspath,
		Dataset: ds,
	}

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

	if pinner, ok := r.repo.Store().(cafs.Pinner); ok {
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
		file  cafs.File
		d     []byte
		store = r.repo.Store()
	)
	ds, err := dsfs.LoadDataset(store, p.Path)
	if err != nil {
		return err
	}

	if p.All {
		file, err = dsfs.LoadData(store, ds)
	} else {
		d, err = dsfs.LoadRows(store, ds, p.Limit, p.Offset)
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
	fs, ok := r.repo.Store().(*ipfs.Filestore)
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

	ds, err := dsfs.LoadDataset(fs, path)
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
