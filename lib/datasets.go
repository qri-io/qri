package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/dsdiff"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// DatasetRequests encapsulates business logic for working with Datasets on Qri
type DatasetRequests struct {
	cli  *rpc.Client
	node *p2p.QriNode
}

// CoreRequestsName implements the Requets interface
func (DatasetRequests) CoreRequestsName() string { return "datasets" }

// NewDatasetRequests creates a DatasetRequests pointer from either a repo
// or an rpc.Client
func NewDatasetRequests(node *p2p.QriNode, cli *rpc.Client) *DatasetRequests {
	if node != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewDatasetRequests"))
	}

	return &DatasetRequests{
		node: node,
		cli:  cli,
	}
}

// List returns this repo's datasets
func (r *DatasetRequests) List(p *ListParams, res *[]repo.DatasetRef) error {
	if r.cli != nil {
		p.RPC = true
		return r.cli.Call("DatasetRequests.List", p, res)
	}

	ds := &repo.DatasetRef{
		Peername:  p.Peername,
		ProfileID: p.ProfileID,
	}

	// ensure valid limit value
	if p.Limit <= 0 {
		p.Limit = 25
	}
	// ensure valid offset value
	if p.Offset < 0 {
		p.Offset = 0
	}

	replies, err := actions.ListDatasets(r.node, ds, p.Limit, p.Offset, p.RPC, p.Published)

	*res = replies
	return err
}

// Get retrieves a dataset head (commit, structure, meta, etc) for a given reference, either
// from the local repo or by asking peers for it. The res parameter will be populated upon success.
func (r *DatasetRequests) Get(ref *repo.DatasetRef, res *repo.DatasetRef) error {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Get", ref, res)
	}

	// Handle `qri use` to get the current default dataset.
	if err := DefaultSelectedRef(r.node.Repo, ref); err != nil {
		return err
	}

	if err := actions.DatasetHead(r.node, ref); err != nil {
		return err
	}

	*res = *ref
	return nil
}

// SaveParams encapsulates arguments to Init & Save
type SaveParams struct {
	Dataset    *dataset.DatasetPod // dataset to create
	FilePath   string              // absolute path to dataset file if provided, or empty string
	Private    bool                // option to make dataset private. private data is not currently implimented, see https://github.com/qri-io/qri/issues/291 for updates
	Publish    bool
	DryRun     bool
	ReturnBody bool // if true, res.Dataset.Body will be a cafs.file of the body
}

// Save adds a history entry, updating a dataset
// TODO - need to make sure users aren't forking by referencing commits other than tip
// TODO - currently, if a user adds metadata or structure, but does not add
// data, we load the data from the previous commit
// this means that the SAME data is getting saved to the store
// this could be better/faster by just not reading the data:
// should amend dsfs.CreateDataset to compare the data being added,
// and not add if the hash already exists
// but still use the hash to add to dataset.BodyPath
func (r *DatasetRequests) Save(p *SaveParams, res *repo.DatasetRef) (err error) {
	if r.cli != nil {
		if p.ReturnBody {
			// can't send an io.Reader interface over RPC
			p.ReturnBody = false
			log.Error("cannot return body bytes over RPC, disabling body return")
		}
		return r.cli.Call("DatasetRequests.Save", p, res)
	}

	if p.Private {
		return fmt.Errorf("option to make dataset private not yet implimented, refer to https://github.com/qri-io/qri/issues/291 for updates")
	}
	if p.Dataset == nil {
		return fmt.Errorf("dataset is required")
	}
	if p.FilePath != "" {
		loadDs := &dataset.DatasetPod{}
		f, err := os.Open(p.FilePath)
		if err != nil {
			return err
		}

		fileExt := strings.ToLower(filepath.Ext(p.FilePath))
		switch fileExt {
		case ".yaml", ".yml":
			data, err := ioutil.ReadAll(f)
			if err != nil {
				return err
			}
			if err = dsutil.UnmarshalYAMLDatasetPod(data, loadDs); err != nil {
				return err
			}
		case ".json":
			if err = json.NewDecoder(f).Decode(loadDs); err != nil {
				return err
			}
		default:
			return fmt.Errorf("error, unrecognized file extension: \"%s\"", fileExt)
		}

		// Copy fields from p.Dataset, they were assigned with user-specified options.
		// NOTE: This is the `Assign` pattern.
		if p.Dataset.Name != "" {
			loadDs.Name = p.Dataset.Name
		}
		if p.Dataset.Peername != "" {
			loadDs.Peername = p.Dataset.Peername
		}
		if loadDs.Commit == nil {
			loadDs.Commit = &dataset.CommitPod{}
		}
		if p.Dataset.Commit.Message != "" {
			loadDs.Commit.Message = p.Dataset.Commit.Message
		}
		if p.Dataset.Commit.Title != "" {
			loadDs.Commit.Title = p.Dataset.Commit.Title
		}
		if p.Dataset.BodyPath != "" {
			loadDs.BodyPath = p.Dataset.BodyPath
		}
		if p.Dataset.Transform != nil {
			loadDs.Transform = p.Dataset.Transform
		}
		// Replace the dataset with the merged result.
		p.Dataset = loadDs
	}

	ref, body, err := actions.SaveDataset(r.node, p.Dataset, p.DryRun, true)
	if err != nil {
		log.Debugf("create ds error: %s\n", err.Error())
		return err
	}
	// ref.Dataset = p.Dataset.Encode()

	if p.Publish {
		var done bool
		if err = NewRegistryRequests(r.node, nil).Publish(&PublishParams{Ref: ref, Pin: true}, &done); err != nil {
			return err
		}
	}

	if p.ReturnBody {
		res.Dataset.Body = body
	}

	*res = ref
	return nil
}

// SetPublishStatus updates the publicity of a reference in the peer's namespace
func (r *DatasetRequests) SetPublishStatus(ref *repo.DatasetRef, res *bool) error {
	res = &ref.Published
	return actions.SetPublishStatus(r.node, ref, ref.Published)
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

	if p.Current.IsEmpty() {
		return fmt.Errorf("current name is required to rename a dataset")
	}

	if err := actions.RenameDataset(r.node, &p.Current, &p.New); err != nil {
		return err
	}

	if err = actions.DatasetHead(r.node, &p.New); err != nil {
		log.Debug(err.Error())
		return err
	}
	*res = p.New
	return nil
}

// Remove a dataset
func (r *DatasetRequests) Remove(p *repo.DatasetRef, ok *bool) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Remove", p, ok)
	}

	if p.Path == "" && (p.Peername == "" && p.Name == "") {
		return fmt.Errorf("either peername/name or path is required")
	}

	if err = actions.DeleteDataset(r.node, p); err != nil {
		return
	}

	// if rc := r.Registry(); rc != nil {
	// 	dse := ds.Encode()
	// 	// TODO - this should be set by LoadDataset
	// 	dse.Path = ref.Path
	// 	if e := rc.DeleteDataset(ref.Peername, ref.Name, dse, pro.PrivKey.GetPublic()); e != nil {
	// 		// ignore registry errors
	// 		log.Errorf("deleting dataset: %s", e.Error())
	// 	}
	// }

	*ok = true
	return nil
}

// LookupParams defines parameters for looking up the body of a dataset
type LookupParams struct {
	Format        dataset.DataFormat
	FormatConfig  dataset.FormatConfig
	Path          string
	Limit, Offset int
	All           bool
}

// LookupResult combines data with it's hashed path
type LookupResult struct {
	Path string `json:"path"`
	// TODO: Rename to Body
	Data []byte `json:"data"`
}

// LookupBody retrieves the dataset body
func (r *DatasetRequests) LookupBody(p *LookupParams, data *LookupResult) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.StructuredData", p, data)
	}

	if p.Limit < 0 || p.Offset < 0 {
		return fmt.Errorf("invalid limit / offset settings")
	}

	bodyPath, bufData, err := actions.LookupBody(r.node, p.Path, p.Format, p.FormatConfig, p.Limit, p.Offset, p.All)
	if err != nil {
		return err
	}

	*data = LookupResult{
		Path: bodyPath,
		Data: bufData,
	}
	return nil
}

// Add adds an existing dataset to a peer's repository
func (r *DatasetRequests) Add(ref *repo.DatasetRef, res *repo.DatasetRef) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Add", ref, res)
	}

	err = actions.AddDataset(r.node, ref)
	*res = *ref
	return err
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

	if err = DefaultSelectedRef(r.node.Repo, &p.Ref); err != nil {
		return
	}

	// TODO: restore validating data from a URL
	// if p.URL != "" && ref.IsEmpty() && o.Schema == nil {
	//   return (lib.NewError(ErrBadArgs, "if you are validating data from a url, please include a dataset name or supply the --schema flag with a file path that Qri can validate against"))
	// }
	if p.Ref.IsEmpty() && p.Data == nil && p.Schema == nil {
		// err = fmt.Errorf("please provide a dataset name, or a supply the --body and --schema flags with file paths")
		return NewError(ErrBadArgs, "please provide a dataset name, or a supply the --body and --schema flags with file paths")
	}

	var body, schema cafs.File
	if p.Data != nil {
		body = cafs.NewMemfileReader(p.DataFilename, p.Data)
	}
	if p.Schema != nil {
		schema = cafs.NewMemfileReader("schema.json", p.Schema)
	}

	// var body cafs.File
	// TODO - restore
	*errors, err = actions.Validate(r.node, p.Ref, body, schema)
	return
}

// DiffParams defines parameters for diffing two datasets with Diff
type DiffParams struct {
	// The pointers to the datasets to diff
	Left, Right repo.DatasetRef
	// override flag to diff full dataset without having to specify each component
	DiffAll bool
	// if DiffAll is false, DiffComponents specifies which components of a dataset to diff
	// currently supported components include "structure", "data", "meta", "transform", and "viz"
	DiffComponents map[string]bool
}

// Diff computes the diff of two datasets
func (r *DatasetRequests) Diff(p *DiffParams, diffs *map[string]*dsdiff.SubDiff) (err error) {
	refs := []repo.DatasetRef{}

	// Handle `qri use` to get the current default dataset.
	if err := DefaultSelectedRefs(r.node.Repo, &refs); err != nil {
		return err
	}
	// fill in the left side if Left is empty, and there are enough
	// refs in the `use` list
	if p.Left.IsEmpty() && len(refs) > 0 {
		p.Left = refs[0]
	}
	// fill in the right side if Right is empty, and there are enough
	// refs in the `use` list
	if p.Right.IsEmpty() && len(refs) > 1 {
		p.Right = refs[1]
	}

	*diffs, err = actions.DiffDatasets(r.node, p.Left, p.Right, p.DiffAll, p.DiffComponents)
	return
}
