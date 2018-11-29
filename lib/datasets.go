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
	"github.com/qri-io/qri/manifest"
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

// SaveParams encapsulates arguments to Save
type SaveParams struct {
	// dataset to create if both Dataset and DatasetPath are provided
	// dataset values will override any values in the document at DatasetPath
	Dataset *dataset.DatasetPod
	// absolute path or URL to a dataset file to load dataset from
	DatasetPath string
	// secrets for transform execution
	Secrets map[string]string
	// option to make dataset private. private data is not currently implimented,
	// see https://github.com/qri-io/qri/issues/291 for updates
	Private bool
	// if true, set saved dataset to published
	Publish bool
	// run without saving, returning results
	DryRun bool
	// if true, res.Dataset.Body will be a cafs.file of the body
	ReturnBody bool
	// if true, convert body to the format of the previous version, if applicable
	ConvertFormatToPrev bool
	// string of references to recall before saving
	Recall string
}

// Save adds a history entry, updating a dataset
// TODO - need to make sure users aren't forking by referencing commits other than tip
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

	ds := p.Dataset
	if ds == nil && p.DatasetPath == "" {
		return fmt.Errorf("at least one of Dataset, DatasetPath is required")
	}

	if p.Recall != "" {
		ref := repo.DatasetRef{
			Peername: ds.Peername,
			Name:     ds.Name,
			// TODO - fix, but really this should be fine for a while because
			// ProfileID is required to be local when saving
			// ProfileID: ds.ProfileID,
			Path: ds.Path,
		}
		recall, err := actions.Recall(r.node, p.Recall, ref)
		if err != nil {
			return err
		}
		recall.Assign(ds)
		ds = recall
	}

	if p.DatasetPath != "" {
		dsf, err := ReadDatasetFile(p.DatasetPath)
		if err != nil {
			return err
		}
		dsf.Assign(ds)
		ds = dsf
	}

	if ds.Name == "" {
		return fmt.Errorf("name is required")
	}
	if ds.BodyPath == "" && ds.Body == nil && ds.BodyBytes == nil && ds.Structure == nil && ds.Meta == nil && ds.Viz == nil && ds.Transform == nil {
		return fmt.Errorf("no changes to save")
	}

	ref, body, err := actions.SaveDataset(r.node, ds, p.Secrets, p.DryRun, true, p.ConvertFormatToPrev)
	if err != nil {
		log.Debugf("create ds error: %s\n", err.Error())
		return err
	}
	// TODO - check to make sure RPC saves aren't horribly broken, and if not remove this
	// ref.Dataset = p.Dataset.Encode()

	if p.Publish {
		var done bool
		if err = NewRegistryRequests(r.node, nil).Publish(&PublishParams{Ref: ref, Pin: true}, &done); err != nil {
			return err
		}
	}

	if p.ReturnBody && ref.Dataset != nil {
		ref.Dataset.Body = body
	}

	*res = ref
	return nil
}

// UpdateParams defines parameters for the Update command
type UpdateParams struct {
	Ref        string
	Title      string
	Message    string
	Recall     string
	Secrets    map[string]string
	Publish    bool
	DryRun     bool
	ReturnBody bool
}

// Update advances a dataset to the latest known version from either a peer or by
// re-running a transform in the peer's namespace
func (r *DatasetRequests) Update(p *UpdateParams, res *repo.DatasetRef) error {
	if r.cli != nil {
		if p.ReturnBody {
			// can't send an io.Reader interface over RPC
			p.ReturnBody = false
			log.Error("cannot return body bytes over RPC, disabling body return")
		}
		return r.cli.Call("DatasetRequests.Update", p, res)
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}

	ref.Dataset = &dataset.DatasetPod{
		Commit: &dataset.CommitPod{
			Title:   p.Title,
			Message: p.Message,
		},
		Transform: &dataset.TransformPod{
			Secrets: p.Secrets,
		},
	}

	if p.Recall != "" {
		ref := repo.DatasetRef{
			Peername: ref.Peername,
			Name:     ref.Name,
			// TODO - fix, but really this should be fine for a while because
			// ProfileID is required to be local when saving
			// ProfileID: ds.ProfileID,
			Path: ref.Path,
		}
		recall, err := actions.Recall(r.node, p.Recall, ref)
		if err != nil {
			return err
		}
		// only transform is assignable
		ref.Dataset.Transform.Assign(recall.Transform)
	}

	result, body, err := actions.UpdateDataset(r.node, &ref, p.Secrets, p.DryRun, true)
	if err != nil {
		return err
	}
	if p.ReturnBody {
		result.Dataset.Body = body
	}
	*res = result

	return nil
}

// SetPublishStatusParams encapsulates parameters for setting the publication status of a dataset
type SetPublishStatusParams struct {
	Ref            *repo.DatasetRef
	UpdateRegistry bool
}

// SetPublishStatus updates the publicity of a reference in the peer's namespace
func (r *DatasetRequests) SetPublishStatus(p *SetPublishStatusParams, res *bool) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.SetPublishStatus", p, res)
	}

	ref := p.Ref
	res = &ref.Published
	if err = actions.SetPublishStatus(r.node, ref, ref.Published); err != nil {
		return err
	}

	if p.UpdateRegistry && r.node.Repo.Registry() != nil {
		if ref.Published == true {
			if err = actions.Publish(r.node, *ref); err != nil {
				return err
			}
			if err = actions.Pin(r.node, *ref); err != nil {
				return err
			}
		} else {
			if err = actions.Unpublish(r.node, *ref); err != nil {
				return err
			}
			if err = actions.Unpin(r.node, *ref); err != nil {
				return err
			}
		}
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

// ValidateDatasetParams defines parameters for dataset
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

// Manifest generates a manifest for a dataset path
func (r *DatasetRequests) Manifest(refstr *string, m *manifest.Manifest) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Manifest", refstr, m)
	}

	ref, err := repo.ParseDatasetRef(*refstr)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(r.node.Repo, &ref); err != nil {
		return
	}

	var mf *manifest.Manifest
	mf, err = actions.NewManifest(r.node, ref.Path)
	if err != nil {
		return
	}
	*m = *mf
	return
}

// ManifestMissing generates a manifest of blocks that are not present on this repo for a given manifest
func (r *DatasetRequests) ManifestMissing(a, b *manifest.Manifest) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Manifest", a, b)
	}

	var mf *manifest.Manifest
	mf, err = actions.Missing(r.node, a)
	if err != nil {
		return
	}
	*b = *mf
	return
}
