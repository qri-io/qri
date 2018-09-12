package lib

import (
	"fmt"
	"io"
	"net/rpc"

	"github.com/qri-io/cafs"

	"github.com/qri-io/dataset"
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

// Repo exposes the DatasetRequest's repo
// TODO - this is an architectural flaw resulting from not having a clear
// order of local > network > RPC requests figured out
// func (r *DatasetRequests) Repo() repo.Repo {
// 	return r.repo
// }

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

	replies, err := actions.ListDatasets(r.node, ds, p.Limit, p.Offset, p.RPC)

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
		return nil
	}

	*res = *ref
	return nil
}

// SaveParams encapsulates arguments to Init & Save
type SaveParams struct {
	Dataset *dataset.DatasetPod // dataset to create
	Private bool                // option to make dataset private. private data is not currently implimented, see https://github.com/qri-io/qri/issues/291 for updates
	Publish bool
}

// New creates a new qri dataset from a source of data
func (r *DatasetRequests) New(p *SaveParams, res *repo.DatasetRef) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.New", p, res)
	}

	if p.Private {
		return fmt.Errorf("option to make dataset private not yet implimented, refer to https://github.com/qri-io/qri/issues/291 for updates")
	}

	ds, bodyFile, secrets, err := actions.NewDataset(p.Dataset)
	if err != nil {
		return err
	}

	*res, err = actions.CreateDataset(r.node, p.Dataset.Name, ds, bodyFile, secrets, true)
	if err != nil {
		log.Debugf("error creating dataset: %s\n", err.Error())
		return err
	}

	if p.Publish {
		var done bool

		if err = NewRegistryRequests(r.node, nil).Publish(&PublishParams{Ref: *res, Pin: true}, &done); err != nil {
			return err
		}
	}

	return actions.ReadDataset(r.node.Repo, res)
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
		return r.cli.Call("DatasetRequests.Save", p, res)
	}

	if p.Private {
		return fmt.Errorf("option to make dataset private not yet implimented, refer to https://github.com/qri-io/qri/issues/291 for updates")
	}

	ds, body, secrets, err := actions.UpdateDataset(r.node, p.Dataset)
	if err != nil {
		return err
	}

	ref, err := actions.CreateDataset(r.node, p.Dataset.Name, ds, body, secrets, true)
	if err != nil {
		log.Debugf("create ds error: %s\n", err.Error())
		return err
	}
	ref.Dataset = ds.Encode()

	if p.Publish {
		var done bool
		if err = NewRegistryRequests(r.node, nil).Publish(&PublishParams{Ref: ref, Pin: true}, &done); err != nil {
			return err
		}
	}

	*res = ref
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

	if p.Current.IsEmpty() {
		return fmt.Errorf("current name is required to rename a dataset")
	}

	if err := actions.RenameDataset(r.node.Repo, p.Current, p.New); err != nil {
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

	if err = actions.DeleteDataset(r.node, *p); err != nil {
		return
	}

	// if pinner, ok := r.repo.Store().(cafs.Pinner); ok {
	// 	// path := datastore.NewKey(strings.TrimSuffix(p.Path, "/"+dsfs.PackageFileDataset.String()))
	// 	if err = pinner.Unpin(datastore.NewKey(p.Path), true); err != nil {
	// 		log.Debug(err.Error())
	// 		return
	// 	}
	// }

	// if err = r.repo.DeleteRef(*p); err != nil {
	// 	log.Debug(err.Error())
	// 	return
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

	bufData, err := actions.LookupBody(r.node, p.Path, p.Format, p.FormatConfig, p.Limit, p.Offset, p.All)
	if err != nil {
		return err
	}

	*data = LookupResult{
		Path: p.Path,
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
