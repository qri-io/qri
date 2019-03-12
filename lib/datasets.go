package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/rpc"

	"github.com/ghodss/yaml"
	"github.com/qri-io/dag"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/rev"
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

	replies, err := actions.ListDatasets(r.node, ds, p.Limit, p.Offset, p.RPC, p.Published, p.ShowNumVersions)

	*res = replies
	return err
}

// GetParams defines parameters for looking up the body of a dataset
type GetParams struct {
	// Path to get, this will often be a dataset reference like me/dataset
	Path string

	Format       string
	FormatConfig dataset.FormatConfig

	Selector string

	Concise       bool
	Limit, Offset int
	All           bool
}

// GetResult combines data with it's hashed path
type GetResult struct {
	Dataset *dataset.Dataset `json:"data"`
	Bytes   []byte           `json:"bytes"`
}

// Get retrieves datasets and components for a given reference. If p.Ref is provided, it is
// used to load the dataset, otherwise p.Path is parsed to create a reference. The
// dataset will be loaded from the local repo if available, or by asking peers for it.
// Using p.Selector will control what components are returned in res.Bytes. The default,
// a blank selector, will also fill the entire dataset at res.Data. If the selector is "body"
// then res.Bytes is loaded with the body.
func (r *DatasetRequests) Get(p *GetParams, res *GetResult) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Get", p, res)
	}
	ref := &repo.DatasetRef{}

	if p.Path == "" {
		// Handle `qri use` to get the current default dataset.
		if err = DefaultSelectedRef(r.node.Repo, ref); err != nil {
			return
		}
	} else {
		*ref, err = repo.ParseDatasetRef(p.Path)
		if err != nil {
			return fmt.Errorf("'%s' is not a valid dataset reference", p.Path)
		}
	}
	if err = repo.CanonicalizeDatasetRef(r.node.Repo, ref); err != nil {
		return
	}

	ds, err := dsfs.LoadDataset(r.node.Repo.Store(), ref.Path)
	if err != nil {
		return fmt.Errorf("error loading dataset")
	}
	ds.Name = ref.Name
	ds.Peername = ref.Peername
	res.Dataset = ds

	if err = base.OpenDataset(r.node.Repo.Filesystem(), ds); err != nil {
		return
	}

	if p.Selector == "body" {
		// `qri get body` loads the body
		if !p.All && (p.Limit < 0 || p.Offset < 0) {
			return fmt.Errorf("invalid limit / offset settings")
		}
		df, err := dataset.ParseDataFormatString(p.Format)
		if err != nil {
			return err
		}

		bufData, err := actions.GetBody(r.node, ds, df, p.FormatConfig, p.Limit, p.Offset, p.All)
		if err != nil {
			return err
		}

		res.Bytes = bufData
		return err
	} else if p.Selector == "transform.script" && ds.Transform != nil && ds.Transform.ScriptFile() != nil {
		// `qri get transform.script` loads the transform script, as a special case
		// TODO (b5): this is a hack that should be generalized
		res.Bytes, err = ioutil.ReadAll(ds.Transform.ScriptFile())
		return err
	} else if p.Selector == "viz.script" && ds.Viz != nil && ds.Viz.ScriptFile() != nil {
		// `qri get viz.script` loads the visualization script, as a special case
		res.Bytes, err = ioutil.ReadAll(ds.Viz.ScriptFile())
		return err
	} else {
		var value interface{}
		if p.Selector == "" {
			// `qri get` without a selector loads only the dataset head
			value = res.Dataset
		} else {
			// `qri get <selector>` loads only the applicable component / field
			value, err = base.ApplyPath(res.Dataset, p.Selector)
			if err != nil {
				return err
			}
		}
		switch p.Format {
		case "json":
			if p.Concise {
				res.Bytes, err = json.Marshal(value)
			} else {
				res.Bytes, err = json.MarshalIndent(value, "", " ")
			}
		default:
			res.Bytes, err = yaml.Marshal(value)
		}
		return err
	}
}

// SaveParams encapsulates arguments to Save
type SaveParams struct {
	// dataset to create. If both Dataset and FilePath are provided
	// dataset values will override any values in the document at FilePath
	Dataset *dataset.Dataset
	// absolute path or URL to a dataset file or component to load
	FilePath string
	// secrets for transform execution
	Secrets map[string]string
	// option to make dataset private. private data is not currently implimented,
	// see https://github.com/qri-io/qri/issues/291 for updates
	Private bool
	// if true, set saved dataset to published
	Publish bool
	// run without saving, returning results
	DryRun bool
	// if true, res.Dataset.Body will be a fs.file of the body
	ReturnBody bool
	// if true, convert body to the format of the previous version, if applicable
	ConvertFormatToPrev bool
	// string of references to recall before saving
	Recall string
	// force a new commit, even if no changes are detected
	Force bool
	// optional writer to have transform script record standard output to
	// note: this won't work over RPC, only on local calls
	ScriptOutput io.Writer
}

// Save adds a history entry, updating a dataset
// TODO - need to make sure users aren't forking by referencing commits other than tip
func (r *DatasetRequests) Save(p *SaveParams, res *repo.DatasetRef) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Save", p, res)
	}

	if p.Private {
		return fmt.Errorf("option to make dataset private not yet implimented, refer to https://github.com/qri-io/qri/issues/291 for updates")
	}

	ds := p.Dataset
	if ds == nil && p.FilePath == "" {
		return fmt.Errorf("at least one of Dataset, FilePath is required")
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

	if p.FilePath != "" {
		// TODO (b5): handle this with a qfs.Filesystem
		dsf, err := ReadDatasetFile(p.FilePath)
		if err != nil {
			return err
		}
		dsf.Assign(ds)
		ds = dsf
	}

	if ds.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !p.Force &&
		ds.BodyPath == "" &&
		ds.Body == nil &&
		ds.BodyBytes == nil &&
		ds.Structure == nil &&
		ds.Meta == nil &&
		ds.Viz == nil &&
		ds.Transform == nil {
		return fmt.Errorf("no changes to save")
	}

	if err = base.OpenDataset(r.node.Repo.Filesystem(), ds); err != nil {
		return
	}

	ref, err := actions.SaveDataset(r.node, ds, p.Secrets, p.ScriptOutput, p.DryRun, true, p.ConvertFormatToPrev, p.Force)
	if err != nil {
		log.Debugf("create ds error: %s\n", err.Error())
		return err
	}

	if p.ReturnBody {
		if err = base.InlineJSONBody(ref.Dataset); err != nil {
			return err
		}
	}

	if p.Publish {
		var publishedRef repo.DatasetRef
		err = r.SetPublishStatus(&SetPublishStatusParams{
			Ref:               ref.String(),
			PublishStatus:     true,
			UpdateRegistry:    true,
			UpdateRegistryPin: true,
		}, &publishedRef)

		if err != nil {
			return err
		}
	}

	*res = ref
	return nil
}

// UpdateParams defines parameters for the Update command
// TODO (b5): I think we can merge this into SaveParams
type UpdateParams struct {
	Ref        string
	Title      string
	Message    string
	Recall     string
	Secrets    map[string]string
	Publish    bool
	DryRun     bool
	ReturnBody bool
	// optional writer to have transform script record standard output to
	// note: this won't work over RPC, only on local calls
	ScriptOutput io.Writer
}

// Update advances a dataset to the latest known version from either a peer or by
// re-running a transform in the peer's namespace
func (r *DatasetRequests) Update(p *UpdateParams, res *repo.DatasetRef) error {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Update", p, res)
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}

	if err = repo.CanonicalizeDatasetRef(r.node.Repo, &ref); err == repo.ErrNotFound {
		return fmt.Errorf("unknown dataset '%s'. please add before updating", ref.AliasString())
	} else if err != nil {
		return err
	}

	if !base.InLocalNamespace(r.node.Repo, &ref) {
		*res, err = actions.UpdateRemoteDataset(r.node, &ref, true)
		return err
	}

	// default to recalling transfrom scripts for local updates
	// TODO (b5): not sure if this should be here or in client libraries
	if p.Recall == "" {
		p.Recall = "tf"
	}

	saveParams := &SaveParams{
		Dataset: &dataset.Dataset{
			Name:      ref.Name,
			Peername:  ref.Peername,
			ProfileID: ref.ProfileID.String(),
			Path:      ref.Path,
			Commit: &dataset.Commit{
				Title:   p.Title,
				Message: p.Message,
			},
			Transform: &dataset.Transform{
				Secrets: p.Secrets,
			},
		},
		Recall:       p.Recall,
		Secrets:      p.Secrets,
		Publish:      p.Publish,
		DryRun:       p.DryRun,
		ReturnBody:   p.ReturnBody,
		ScriptOutput: p.ScriptOutput,
	}

	return r.Save(saveParams, res)
}

// SetPublishStatusParams encapsulates parameters for setting the publication status of a dataset
type SetPublishStatusParams struct {
	Ref               string
	PublishStatus     bool
	UpdateRegistry    bool
	UpdateRegistryPin bool
}

// SetPublishStatus updates the publicity of a reference in the peer's namespace
func (r *DatasetRequests) SetPublishStatus(p *SetPublishStatusParams, publishedRef *repo.DatasetRef) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.SetPublishStatus", p, publishedRef)
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(r.node.Repo, &ref); err != nil {
		return err
	}

	ref.Published = p.PublishStatus
	if err = actions.SetPublishStatus(r.node, &ref, ref.Published); err != nil {
		return err
	}

	*publishedRef = ref

	if p.UpdateRegistry && r.node.Repo.Registry() != nil {
		var done bool
		rr := NewRegistryRequests(r.node, nil)

		if ref.Published {
			if err = rr.Publish(&ref, &done); err != nil {
				return
			}

			if p.UpdateRegistryPin {
				return rr.Pin(&ref, &done)
			}
		} else {
			if err = rr.Unpublish(&ref, &done); err != nil {
				return
			}

			if p.UpdateRegistryPin {
				return rr.Unpin(&ref, &done)
			}
		}
	}

	return
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

	if err := actions.ModifyDataset(r.node, &p.Current, &p.New, true /*isRename*/); err != nil {
		return err
	}

	if err = actions.DatasetHead(r.node, &p.New); err != nil {
		log.Debug(err.Error())
		return err
	}
	*res = p.New
	return nil
}

// RemoveParams defines parameters for remove command
type RemoveParams struct {
	Ref string
	// Ref      *repo.DatasetRef
	Revision rev.Rev
}

// RemoveResponse gives the results of a remove
type RemoveResponse struct {
	Ref        string
	NumDeleted int
}

// Remove a dataset entirely or remove a certain number of revisions
func (r *DatasetRequests) Remove(p *RemoveParams, res *RemoveResponse) error {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Remove", p, res)
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}

	if err = repo.CanonicalizeDatasetRef(r.node.Repo, &ref); err != nil {
		return err
	}
	res.Ref = ref.String()

	if ref.Path == "" && ref.Peername == "" && ref.Name == "" {
		return fmt.Errorf("either peername/name or path is required")
	}

	if p.Revision.Field != "ds" {
		return fmt.Errorf("can only delete whole dataset revisions, not individual fields")
	}

	if p.Revision.Gen == rev.AllGenerations {
		// Delete entire dataset for all generations.
		if err := actions.DeleteDataset(r.node, &ref); err != nil {
			return err
		}
		res.NumDeleted = rev.AllGenerations
		return nil
	} else if p.Revision.Gen < 1 {
		return fmt.Errorf("invalid number of revisions to delete: %d", p.Revision.Gen)
	}

	// Get the revisions that will be deleted.
	log, err := actions.DatasetLog(r.node, ref, p.Revision.Gen+1, 0)
	if err != nil {
		return err
	}

	// If deleting more revisions then exist, delete the entire dataset.
	if p.Revision.Gen >= len(log) {
		if err := actions.DeleteDataset(r.node, &ref); err != nil {
			return err
		}
		res.NumDeleted = rev.AllGenerations
		return nil
	}

	// Delete the specific number of revisions.
	replace := log[p.Revision.Gen]
	if err := actions.ModifyDataset(r.node, &ref, &replace, false /*isRename*/); err != nil {
		return err
	}
	res.NumDeleted = p.Revision.Gen

	// if rc := r.Registry(); rc != nil {
	// 	dse := ds.Encode()
	// 	// TODO - this should be set by LoadDataset
	// 	dse.Path = ref.Path
	// 	if e := rc.DeleteDataset(ref.Peername, ref.Name, dse, pro.PrivKey.GetPublic()); e != nil {
	// 		// ignore registry errors
	// 		log.Errorf("deleting dataset: %s", e.Error())
	// 	}
	// }

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

	var body, schema qfs.File
	if p.Data != nil {
		body = qfs.NewMemfileReader(p.DataFilename, p.Data)
	}
	if p.Schema != nil {
		schema = qfs.NewMemfileReader("schema.json", p.Schema)
	}

	*errors, err = actions.Validate(r.node, p.Ref, body, schema)
	return
}

// Manifest generates a manifest for a dataset path
func (r *DatasetRequests) Manifest(refstr *string, m *dag.Manifest) (err error) {
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

	var mf *dag.Manifest
	mf, err = actions.NewManifest(r.node, ref.Path)
	if err != nil {
		return
	}
	*m = *mf
	return
}

// ManifestMissing generates a manifest of blocks that are not present on this repo for a given manifest
func (r *DatasetRequests) ManifestMissing(a, b *dag.Manifest) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Manifest", a, b)
	}

	var mf *dag.Manifest
	mf, err = actions.Missing(r.node, a)
	if err != nil {
		return
	}
	*b = *mf
	return
}

// DAGInfo generates a manifest for a dataset path
func (r *DatasetRequests) DAGInfo(refstr *string, i *dag.Info) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.DAGInfo", refstr, i)
	}

	ref, err := repo.ParseDatasetRef(*refstr)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(r.node.Repo, &ref); err != nil {
		return
	}

	var info *dag.Info
	info, err = actions.NewDAGInfo(r.node, ref.Path)
	if err != nil {
		return
	}
	*i = *info
	return
}
