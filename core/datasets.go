package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/rpc"
	"path/filepath"
	"strings"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/dsdiff"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/varName"
)

// DatasetRequests encapsulates business logic for this node's
// user profile
type DatasetRequests struct {
	repo repo.Repo
	cli  *rpc.Client
	Node *p2p.QriNode
}

// Repo exposes the DatasetRequest's repo
// TODO - this is an architectural flaw resulting from not having a clear
// order of local > network > RPC requests figured out
func (r *DatasetRequests) Repo() repo.Repo {
	return r.repo
}

// CoreRequestsName implements the Requets interface
func (DatasetRequests) CoreRequestsName() string { return "datasets" }

// NewDatasetRequests creates a DatasetRequests pointer from either a repo
// or an rpc.Client
func NewDatasetRequests(r repo.Repo, cli *rpc.Client) *DatasetRequests {
	if r != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewDatasetRequests"))
	}

	return &DatasetRequests{
		repo: r,
		cli:  cli,
	}
}

// List returns this repo's datasets
func (r *DatasetRequests) List(p *ListParams, res *[]repo.DatasetRef) error {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.List", p, res)
	}

	ds := &repo.DatasetRef{
		Peername: p.Peername,
		PeerID:   p.PeerID,
	}

	if ds.Peername == "" && ds.PeerID == "" {
		ds.Peername = "me"
	}

	pro, err := r.repo.Profile()
	if err != nil {
		return fmt.Errorf("error getting profile: %s", err.Error())
	}

	if err := repo.CanonicalizePeer(r.repo, ds); err != nil {
		return fmt.Errorf("error canonicalizing peer: %s", err.Error())
	}

	if ds.Peername != pro.Peername {
		if r.Node == nil {
			return fmt.Errorf("cannot list remote datasets without p2p connection")
		}

		replies, err := r.Node.RequestDatasetsList(ds.Peername)
		*res = replies
		return err
	}

	store := r.repo.Store()
	// ensure valid limit value
	if p.Limit <= 0 {
		p.Limit = 25
	}
	// ensure valid offset value
	if p.Offset < 0 {
		p.Offset = 0
	}
	replies, err := r.repo.References(p.Limit, p.Offset)
	if err != nil {
		return fmt.Errorf("error getting namespace: %s", err.Error())
	}

	for i, ref := range replies {
		if i >= p.Limit {
			break
		}

		if err := repo.CanonicalizePeer(r.repo, &replies[i]); err != nil {
			return fmt.Errorf("error canonicalizing dataset peername: %s", err.Error())
		}

		ds, err := dsfs.LoadDataset(store, datastore.NewKey(ref.Path))
		if err != nil {
			// try one extra time...
			// TODO - remove this horrible hack
			ds, err = dsfs.LoadDataset(store, datastore.NewKey(ref.Path))
			if err != nil {
				return fmt.Errorf("error loading path: %s, err: %s", ref.Path, err.Error())
			}
		}
		replies[i].Dataset = ds
	}

	*res = replies
	return nil
}

// Get a dataset
func (r *DatasetRequests) Get(p *repo.DatasetRef, res *repo.DatasetRef) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Get", p, res)
	}

	err = repo.CanonicalizeDatasetRef(r.repo, p)
	if err != nil {
		return err
	}

	getRemote := func(err error) error {
		if r.Node != nil {
			ref, err := r.Node.RequestDatasetInfo(p)
			if ref != nil {
				ds := ref.Dataset
				// TODO - this is really stupid, p2p.RequestDatasetInfo should return an error here
				if ds.IsEmpty() {
					return fmt.Errorf("dataset not found")
				}
				st := ds.Structure
				// TODO - it seems that jsonschema.RootSchema and encoding/gob
				// really don't like each other (no surprise there, thanks to validators being an interface)
				// Which means that when this response is proxied over the wire bad things happen
				// So for now we're doing this weirdness with re-creating a gob-friendly version
				// of a dataset
				*res = repo.DatasetRef{
					Peername: p.Peername,
					Name:     p.Name,
					Path:     ref.Path,
					Dataset: &dataset.Dataset{
						Commit: ds.Commit,
						Meta:   ds.Meta,
						Structure: &dataset.Structure{
							Checksum:     st.Checksum,
							Compression:  st.Compression,
							Encoding:     st.Encoding,
							ErrCount:     st.ErrCount,
							Entries:      st.Entries,
							Format:       st.Format,
							FormatConfig: st.FormatConfig,
							Length:       st.Length,
							Qri:          st.Qri,
						},
					},
				}
			}
			return err
		}
		return err
	}

	store := r.repo.Store()
	ds, err := dsfs.LoadDataset(store, datastore.NewKey(p.Path))
	if err != nil {
		return getRemote(err)
	}

	*res = repo.DatasetRef{
		PeerID:   p.PeerID,
		Peername: p.Peername,
		Name:     p.Name,
		Path:     p.Path,
		Dataset:  ds,
	}
	return nil
}

// InitParams encapsulates arguments to Init
type InitParams struct {
	Peername          string    // name of peer creating this dataset. required.
	Name              string    // variable name for referring to this dataset. required.
	URL               string    // url to download data from. either Url or Data is required
	DataFilename      string    // filename of data file. extension is used for filetype detection
	Data              io.Reader // reader of structured data. either Url or Data is required
	MetadataFilename  string    // filename of metadata file. optional.
	Metadata          io.Reader // reader of json-formatted metadata
	StructureFilename string    // filename of metadata file. optional.
	Structure         io.Reader // reader of json-formatted metadata
	Private           bool      // option to make dataset private. private data is not currently implimented, see https://github.com/qri-io/qri/issues/291 for updates
}

// Init creates a new qri dataset from a source of data
func (r *DatasetRequests) Init(p *InitParams, res *repo.DatasetRef) error {
	if p.Private {
		return fmt.Errorf("option to make dataset private not available yet, refer to https://github.com/qri-io/qri/issues/291 for updates")
	}

	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Init", p, res)
	}

	var (
		rdr      io.Reader
		store    = r.repo.Store()
		filename = p.DataFilename
	)

	if p.URL != "" {
		res, err := http.Get(p.URL)
		if err != nil {
			return fmt.Errorf("error fetching url: %s", err.Error())
		}
		filename = filepath.Base(p.URL)
		defer res.Body.Close()
		rdr = res.Body
	} else if p.Data != nil {
		rdr = p.Data
	} else {
		return fmt.Errorf("either a file or a url is required to create a dataset")
	}

	if p.Name != "" {
		if err := validate.ValidName(p.Name); err != nil {
			return fmt.Errorf("invalid name: %s", err.Error())
		}
	}

	// TODO - need a better strategy for huge files
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		return fmt.Errorf("error reading file: %s", err.Error())
	}

	// read structure from InitParams, or detect from data
	st := &dataset.Structure{}
	if p.Structure != nil {
		if err := json.NewDecoder(p.Structure).Decode(st); err != nil {
			return fmt.Errorf("error parsing structure json: %s", err.Error())
		}
	} else {
		st, err = detect.FromReader(filename, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("error determining dataset schema: %s", err.Error())
		}
	}

	// Ensure that dataset contains valid field names
	if err = validate.Structure(st); err != nil {
		return fmt.Errorf("invalid structure: %s", err.Error())
	}

	// TODO - restore
	// if err := validate.DataFormat(st.Format, bytes.NewReader(data)); err != nil {
	// 	return fmt.Errorf("invalid data format: %s", err.Error())
	// }

	// TODO - check for errors in dataset and warn user if errors exist

	datakey, err := store.Put(cafs.NewMemfileBytes("data."+st.Format.String(), data), false)
	if err != nil {
		return fmt.Errorf("error putting data file in store: %s", err.Error())
	}

	dataexists, err := repo.HasPath(r.repo, datakey)
	if err != nil && !strings.Contains(err.Error(), repo.ErrRepoEmpty.Error()) {
		return fmt.Errorf("error checking repo for already-existing data: %s", err.Error())
	}
	if dataexists {
		return fmt.Errorf("this data already exists")
	}

	name := p.Name
	if name == "" && filename != "" {
		name = varName.CreateVarNameFromString(filename)
	}

	ds := &dataset.Dataset{
		Meta:      &dataset.Meta{},
		Commit:    &dataset.Commit{Title: "initial commit"},
		Structure: st,
	}
	if p.Metadata != nil {
		if err := json.NewDecoder(p.Metadata).Decode(ds.Meta); err != nil {
			return fmt.Errorf("error parsing metadata json: %s", err.Error())
		}
	}
	if p.URL != "" {
		ds.Meta.DownloadPath = p.URL
		// if we're adding from a dataset url, set a default accrual periodicity of once a week
		// this'll set us up to re-check urls over time
		// TODO - make this configurable via a param?
		ds.Meta.AccrualPeriodicity = "R/P1W"
	}

	dataf := cafs.NewMemfileBytes("data."+st.Format.String(), data)
	dskey, err := r.repo.CreateDataset(ds, dataf, true)
	if err != nil {
		fmt.Printf("error creating dataset: %s\n", err.Error())
		return err
	}

	ref := repo.DatasetRef{Peername: p.Peername, Name: name, Path: dskey.String(), Dataset: ds}

	if err := repo.CanonicalizePeer(r.repo, &ref); err != nil {
		return fmt.Errorf("error canonicalizing peername: %s", err.Error())
	}

	if err = r.repo.PutRef(ref); err != nil {
		return fmt.Errorf("error adding dataset name to repo: %s", err.Error())
	}

	// ds, err = r.repo.GetDataset(dskey)
	// if err != nil {
	// 	return fmt.Errorf("error reading dataset: '%s': %s", dskey.String(), err.Error())
	// }

	*res = ref
	return nil
}

// SaveParams defines permeters for Dataset Saves
type SaveParams struct {
	Name              string    // dataset name
	Peername          string    // peername
	URL               string    // string of url to get new data. optional.
	DataFilename      string    // filename for new data. optional.
	Data              io.Reader // stream of complete dataset update.
	MetadataFilename  string    // filename for new data. optional.
	Metadata          io.Reader // stream of complete dataset update. optional.
	StructureFilename string    // filename for new data. optional.
	Structure         io.Reader // stream of complete dataset update.
	Title             string    // save message title. required.
	Message           string    // save message. optional.
}

// Save adds a history entry, updating a dataset
// TODO - need to make sure users aren't forking by referncing commits other than tip
// TODO - currently, if a user adds metadata or structure, but does not add
// data, we load the data from the previous commit
// this means that the SAME data is getting saved to the store
// this bad!
// should amend dsfs.CreateDataset to compare the data being added,
// and not add if the hash already exists
// but still use the hash to add to dataset.DataPath
func (r *DatasetRequests) Save(p *SaveParams, res *repo.DatasetRef) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Save", p, res)
	}

	var (
		data     []byte
		dataf    cafs.File
		ds       = &dataset.Dataset{}
		store    = r.repo.Store()
		filename = p.DataFilename
	)

	prevReq := &repo.DatasetRef{Name: p.Name, Peername: p.Peername}

	if err = repo.CanonicalizeDatasetRef(r.repo, prevReq); err != nil {
		return fmt.Errorf("error canonicalizing previous dataset reference: %s", err.Error())
	}

	prev := &repo.DatasetRef{}

	if err := r.Get(prevReq, prev); err != nil {
		return fmt.Errorf("error getting previous dataset: %s", err.Error())
	}

	if p.URL == "" && p.Data == nil && p.Metadata == nil && p.Structure == nil {
		return fmt.Errorf("to save update, need a URL or data file, metadata file, or structure file")
	}

	if p.URL != "" && p.Data != nil {
		return fmt.Errorf("to save update, need either a URL or data file")
	}

	if p.URL != "" {
		res, err := http.Get(p.URL)
		if err != nil {
			return fmt.Errorf("error fetching url: %s", err.Error())
		}
		filename = filepath.Base(p.URL)
		defer res.Body.Close()
		p.Data = res.Body
	}

	if p.Data != nil {
		// TODO - need a better strategy for huge files
		data, err = ioutil.ReadAll(p.Data)
		if err != nil {
			return fmt.Errorf("error reading file: %s", err.Error())
		}
	} else {
		// load data cause we need something to compare the structure to
		datafile, err := dsfs.LoadData(store, prev.Dataset)
		if err != nil {
			return fmt.Errorf("error loading previous data from filestore: %s", err)
		}
		// TODO - need a better strategy for huge files
		data, err = ioutil.ReadAll(datafile)
		if err != nil {
			return fmt.Errorf("error reading file after file was loaded from filestore: %s", err)
		}
		// TODO - need a filename with an extension because that is how
		// we determine the schema in line 421 in detect.FromReader
		// however, when reading from IPFS, the datafile.Filename
		// is an ipfs hash, with no extention
		// using this janky way of constructing a fake filename
		// for us to use later when we detect the schema
		filename = "data." + prev.Dataset.Structure.Format.String()
	}

	// read structure from SaveParams, or detect from data
	st := &dataset.Structure{}
	if p.Structure != nil {
		if err := json.NewDecoder(p.Structure).Decode(st); err != nil {
			return fmt.Errorf("error parsing structure json: %s", err.Error())
		}
	} else {
		st, err = detect.FromReader(filename, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("error determining dataset schema: %s", err.Error())
		}
	}

	// read meta from SaveParams, edit to include URL download Path if needed
	mt := &dataset.Meta{}
	if p.Metadata != nil {
		if err := json.NewDecoder(p.Metadata).Decode(mt); err != nil {
			return fmt.Errorf("error parsing metadata json: %s", err.Error())
		}
	}
	if p.URL != "" {
		mt.DownloadPath = p.URL
		// if we're adding from a dataset url, set a default accrual periodicity of once a week
		// this'll set us up to re-check urls over time
		// TODO - make this configurable via a param?
		mt.AccrualPeriodicity = "R/P1W"
	}
	changes := &dataset.Dataset{
		Commit:    &dataset.Commit{Title: p.Title, Message: p.Message},
		Structure: st,
		Meta:      mt,
	}

	// add all previous fields and any changes
	ds.Assign(prev.Dataset, changes)
	ds.PreviousPath = prev.Path

	// ds.Assign clobbers empty commit messages with the previous
	// commit message. So if the peer hasn't provided a message at this point
	// let's maintain that going into CreateDataset
	if p.Title == "" {
		ds.Commit.Title = ""
	}
	if p.Message == "" {
		ds.Commit.Message = ""
	}

	// Assign will assign any previous paths to the current paths
	// the dsdiff (called in dsfs.CreateDataset), will compare the paths
	// see that they are the same, and claim there are no differences
	// since we will potentially have changes in the Meta and Structure
	// we want the differ to have to compare each field
	// so we reset the paths
	ds.Meta.SetPath("")
	ds.Structure.SetPath("")

	dataf = cafs.NewMemfileBytes("data."+st.Format.String(), data)
	dspath, err := r.repo.CreateDataset(ds, dataf, true)
	if err != nil {
		fmt.Printf("create ds error: %s\n", err.Error())
		return err
	}

	if prev.Name != "" {
		if err := r.repo.DeleteRef(*prev); err != nil {
			return err
		}
		prev.Path = dspath.String()
		if err := r.repo.PutRef(*prev); err != nil {
			return err
		}
	}

	*res = repo.DatasetRef{
		Peername: p.Peername,
		Name:     p.Name,
		Path:     dspath.String(),
		Dataset:  ds,
	}

	return nil
}

// RenameParams defines parameters for Dataset renaming
type RenameParams struct {
	Current, New repo.DatasetRef
}

// Rename changes a user's given name for a dataset
func (r *DatasetRequests) Rename(p *RenameParams, res *repo.DatasetRef) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Rename", p, res)
	}

	if err := repo.CanonicalizeDatasetRef(r.repo, &p.Current); err != nil {
		return fmt.Errorf("error canonicalizing existing reference: %s", err.Error())
	}
	if err := repo.CanonicalizeDatasetRef(r.repo, &p.New); err != nil {
		return fmt.Errorf("error canonicalizing new reference: %s", err.Error())
	}

	if p.Current.IsEmpty() {
		return fmt.Errorf("current name is required to rename a dataset")
	}

	if err := validate.ValidName(p.New.Name); err != nil {
		return err
	}

	if _, err := r.repo.GetRef(p.New); err != repo.ErrNotFound {
		return fmt.Errorf("dataset '%s/%s' already exists", p.New.Peername, p.New.Name)
	}

	p.Current, err = r.repo.GetRef(p.Current)
	if err != nil {
		return fmt.Errorf("error getting dataset: %s", err.Error())
	}
	p.New.Path = p.Current.Path
	if err := r.repo.DeleteRef(p.Current); err != nil {
		return err
	}

	if err := r.repo.PutRef(p.New); err != nil {
		return err
	}

	ds, err := dsfs.LoadDataset(r.repo.Store(), datastore.NewKey(p.Current.Path))
	if err != nil {
		return err
	}

	*res = repo.DatasetRef{
		Peername: p.New.Peername,
		Name:     p.New.Name,
		Path:     p.Current.Path,
		Dataset:  ds,
	}
	return nil
}

// Remove a dataset
func (r *DatasetRequests) Remove(p *repo.DatasetRef, ok *bool) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Remove", p, ok)
	}

	if err := repo.CanonicalizeDatasetRef(r.repo, p); err != nil {
		return fmt.Errorf("error canonicalizing new reference: %s", err.Error())
	}

	if p.Path == "" && (p.Peername == "" && p.Name == "") {
		return fmt.Errorf("either peername/name or path is required")
	}

	ref, err := r.repo.GetRef(*p)
	if err != nil {
		return
	}

	if ref.Path != p.Path {
		return fmt.Errorf("given path does not equal most recent dataset path: cannot delete a specific save, can only delete entire dataset history. use `me/dataset_name` to delete entire dataset")
	}

	if pinner, ok := r.repo.Store().(cafs.Pinner); ok {
		// path := datastore.NewKey(strings.TrimSuffix(p.Path, "/"+dsfs.PackageFileDataset.String()))
		if err = pinner.Unpin(datastore.NewKey(p.Path), true); err != nil {
			return
		}
	}

	if err = r.repo.DeleteRef(*p); err != nil {
		return
	}

	*ok = true
	return nil
}

// StructuredDataParams defines parameters for retrieving
// structured data (which is the kind of data datasets contain)
type StructuredDataParams struct {
	Format        dataset.DataFormat
	FormatConfig  dataset.FormatConfig
	Path          string
	Limit, Offset int
	All           bool
}

// StructuredData combines data with it's hashed path
type StructuredData struct {
	Path string `json:"path"`
	Data []byte `json:"data"`
}

// StructuredData retrieves dataset data
func (r *DatasetRequests) StructuredData(p *StructuredDataParams, data *StructuredData) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.StructuredData", p, data)
	}

	var (
		file  cafs.File
		store = r.repo.Store()
		read  = 0
	)

	if p.Limit < 0 || p.Offset < 0 {
		return fmt.Errorf("invalid limit / offset settings")
	}

	ds, err := dsfs.LoadDataset(store, datastore.NewKey(p.Path))
	if err != nil {
		return err
	}

	file, err = dsfs.LoadData(store, ds)
	if err != nil {
		return err
	}

	st := &dataset.Structure{}
	st.Assign(ds.Structure, &dataset.Structure{
		Format:       p.Format,
		FormatConfig: p.FormatConfig,
		Schema:       dataset.BaseSchemaArray,
	})

	buf, err := dsio.NewEntryBuffer(st)
	if err != nil {
		return fmt.Errorf("error allocating result buffer: %s", err)
	}
	rr, err := dsio.NewEntryReader(ds.Structure, file)
	if err != nil {
		return fmt.Errorf("error allocating data reader: %s", err)
	}

	for i := 0; i >= 0; i++ {
		val, err := rr.ReadEntry()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("row iteration error: %s", err.Error())
		}
		if !p.All && i < p.Offset {
			continue
		}
		if err := buf.WriteEntry(val); err != nil {
			return fmt.Errorf("error writing value to buffer: %s", err.Error())
		}
		read++
		if read == p.Limit {
			break
		}
	}

	if err := buf.Close(); err != nil {
		return fmt.Errorf("error closing row buffer: %s", err.Error())
	}

	*data = StructuredData{
		Path: p.Path,
		Data: buf.Bytes(),
	}
	return nil
}

// Add adds an existing dataset to a peer's repository
func (r *DatasetRequests) Add(ref *repo.DatasetRef, res *repo.DatasetRef) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Add", ref, res)
	}

	if err := repo.CanonicalizeDatasetRef(r.repo, ref); err != nil {
		return fmt.Errorf("error canonicalizing new reference: %s", err.Error())
	}

	if ref.Path == "" && r.Node != nil {
		res, err := r.Node.RequestDatasetInfo(ref)
		if err != nil {
			return err
		}
		ref = res
	}

	fs, ok := r.repo.Store().(*ipfs.Filestore)
	if !ok {
		return fmt.Errorf("can only add datasets when running an IPFS filestore")
	}

	key := datastore.NewKey(strings.TrimSuffix(ref.Path, "/"+dsfs.PackageFileDataset.String()))

	_, err = fs.Fetch(cafs.SourceAny, key)
	if err != nil {
		return fmt.Errorf("error fetching file: %s", err.Error())
	}

	err = fs.Pin(key, true)
	if err != nil {
		return fmt.Errorf("error pinning root key: %s", err.Error())
	}

	path := datastore.NewKey(key.String() + "/" + dsfs.PackageFileDataset.String())

	profile, err := r.repo.Profile()
	if err != nil {
		return fmt.Errorf("error getting profile: %s", err)
	}

	ref.Peername = profile.Peername
	ref.PeerID = profile.ID

	err = r.repo.PutRef(*ref)
	if err != nil {
		return fmt.Errorf("error putting dataset name in repo: %s", err.Error())
	}

	ds, err := dsfs.LoadDataset(fs, path)
	if err != nil {
		return fmt.Errorf("error loading newly saved dataset path: %s", path.String())
	}

	ref.Dataset = ds

	*res = *ref
	return
}

// ValidateDatasetParams defines paremeters for dataset
// data validation
type ValidateDatasetParams struct {
	Ref repo.DatasetRef
	// URL          string
	DataFilename string
	Data         io.Reader
	Schema       io.Reader
}

// Validate gives a dataset of errors and issues for a given dataset
func (r *DatasetRequests) Validate(p *ValidateDatasetParams, errors *[]jsonschema.ValError) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Validate", p, errors)
	}

	if p.Ref.IsEmpty() && p.Data == nil {
		return fmt.Errorf("either data or a dataset reference is required")
	}

	if err := repo.CanonicalizeDatasetRef(r.repo, &p.Ref); err != nil {
		return fmt.Errorf("error canonicalizing new reference: %s", err.Error())
	}

	var (
		st   = &dataset.Structure{}
		ref  repo.DatasetRef
		data []byte
	)

	// if a dataset is specified, load it
	if p.Ref.Path != "" {
		err = r.Get(&p.Ref, &ref)
		if err != nil {
			return err
		}

		st = ref.Dataset.Structure
	} else if p.Data == nil {
		return fmt.Errorf("cannot find dataset: %s", p.Ref)
	}

	if p.Data != nil {
		data, err = ioutil.ReadAll(p.Data)
		if err != nil {
			return fmt.Errorf("error reading data: %s", err.Error())
		}

		// if no schema, detect one
		if st.Schema == nil {
			str, e := detect.FromReader(p.DataFilename, bytes.NewBuffer(data))
			if e != nil {
				return e
			}
			st = str
		}
	}

	// if a schema is specified, override with it
	if p.Schema != nil {
		stbytes, err := ioutil.ReadAll(p.Schema)
		if err != nil {
			return err
		}
		sch := &jsonschema.RootSchema{}
		if e := sch.UnmarshalJSON(stbytes); e != nil {
			return fmt.Errorf("error reading schema: %s", e.Error())
		}
		st.Schema = sch
	}

	if data == nil && ref.Dataset != nil {
		f, e := dsfs.LoadData(r.repo.Store(), ref.Dataset)
		if e != nil {
			return fmt.Errorf("error loading dataset data: %s", e.Error())
		}
		data, err = ioutil.ReadAll(f)
		if err != nil {
			return fmt.Errorf("error loading dataset data: %s", err.Error())
		}
	}

	er, err := dsio.NewEntryReader(st, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("error reading data: %s", err.Error())
	}

	*errors, err = validate.EntryReader(er)

	return
}

// DiffParams defines parameters for diffing two datasets with Diff
type DiffParams struct {
	// The pointers to the datasets to diff
	DsLeft, DsRight *dataset.Dataset
	// override flag to diff full dataset without having to specify each component
	DiffAll bool
	// if DiffAll is false, DiffComponents specifies which components of a dataset to diff
	// currently supported components include "structure", "data", "meta", "transform", and "visConfig"
	DiffComponents map[string]bool
}

// Diff computes the diff of two datasets
func (r *DatasetRequests) Diff(p *DiffParams, diffs *map[string]*dsdiff.SubDiff) (err error) {
	diffMap := make(map[string]*dsdiff.SubDiff)
	if p.DiffAll {
		diffMap, err := dsdiff.DiffDatasets(p.DsLeft, p.DsRight, nil)
		if err != nil {
			return fmt.Errorf("error diffing datasets: %s", err.Error())
		}
		// TODO: remove this temporary hack
		if diffMap["data"] == nil || len(diffMap["data"].Deltas()) == 0 {
			// dereference data paths
			// marshal json to []byte
			// call `dsdiff.DiffJSON(a, b)`
		}
		*diffs = diffMap
	} else {
		for k, v := range p.DiffComponents {
			if v {
				switch k {
				case "structure":
					if p.DsLeft.Structure != nil && p.DsRight.Structure != nil {
						structureDiffs, err := dsdiff.DiffStructure(p.DsLeft.Structure, p.DsRight.Structure)
						if err != nil {
							return fmt.Errorf("error diffing %s: %s", k, err.Error())
						}
						diffMap[k] = structureDiffs
					}
				case "data":
					//TODO
					if p.DsLeft.DataPath != "" && p.DsRight.DataPath != "" {
						dataDiffs, err := dsdiff.DiffData(p.DsLeft, p.DsRight)
						if err != nil {
							return fmt.Errorf("error diffing %s: %s", k, err.Error())
						}
						diffMap[k] = dataDiffs
					}
				case "transform":
					if p.DsLeft.Transform != nil && p.DsRight.Transform != nil {
						transformDiffs, err := dsdiff.DiffTransform(p.DsLeft.Transform, p.DsRight.Transform)
						if err != nil {
							return fmt.Errorf("error diffing %s: %s", k, err.Error())
						}
						diffMap[k] = transformDiffs
					}
				case "meta":
					if p.DsLeft.Meta != nil && p.DsRight.Meta != nil {
						metaDiffs, err := dsdiff.DiffMeta(p.DsLeft.Meta, p.DsRight.Meta)
						if err != nil {
							return fmt.Errorf("error diffing %s: %s", k, err.Error())
						}
						diffMap[k] = metaDiffs
					}
				case "visConfig":
					if p.DsLeft.VisConfig != nil && p.DsRight.VisConfig != nil {
						visConfigDiffs, err := dsdiff.DiffVisConfig(p.DsLeft.VisConfig, p.DsRight.VisConfig)
						if err != nil {
							return fmt.Errorf("error diffing %s: %s", k, err.Error())
						}
						diffMap[k] = visConfigDiffs
					}
				}
			}
		}
		*diffs = diffMap

	}
	// Hack to examine data
	if p.DiffAll || p.DiffComponents["data"] == true {
		sd1Params := &StructuredDataParams{
			Format: dataset.JSONDataFormat,
			Path:   p.DsLeft.Path().String(),
		}
		sd2Params := &StructuredDataParams{
			Format: dataset.JSONDataFormat,
			Path:   p.DsRight.Path().String(),
		}
		sd1 := &StructuredData{}
		sd2 := &StructuredData{}
		err := r.StructuredData(sd1Params, sd1)
		if err != nil {
			return fmt.Errorf("error getting structured data: %s", err.Error())
		}
		err = r.StructuredData(sd2Params, sd2)
		if err != nil {
			return fmt.Errorf("error getting structured data: %s", err.Error())
		}

		m1 := &map[string]json.RawMessage{"data": sd1.Data}
		m2 := &map[string]json.RawMessage{"data": sd2.Data}
		dataBytes1, err := json.Marshal(m1)
		if err != nil {
			return fmt.Errorf("error marshaling json: %s", err.Error())
		}
		dataBytes2, err := json.Marshal(m2)
		if err != nil {
			return fmt.Errorf("error marshaling json: %s", err.Error())
		}
		dataDiffs, err := dsdiff.DiffJSON(dataBytes1, dataBytes2, "data")
		if err != nil {
			return fmt.Errorf("error comparing structured data: %s", err.Error())
		}
		diffMap["data"] = dataDiffs
	}
	return nil
}
