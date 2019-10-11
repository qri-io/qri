package lib

import (
	"context"
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
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/dsref"
)

// DatasetRequests encapsulates business logic for working with Datasets on Qri
// TODO (b5): switch to using an Instance instead of separate fields
type DatasetRequests struct {
	// TODO (b5) - remove cli & node fields in favour of inst accessors:
	cli  *rpc.Client
	node *p2p.QriNode
	inst *Instance
}

// CoreRequestsName implements the Requets interface
func (DatasetRequests) CoreRequestsName() string { return "datasets" }

// NewDatasetRequests creates a DatasetRequests pointer from either a repo
// or an rpc.Client
//
// Deprecated. use NewDatasetRequestsInstance
func NewDatasetRequests(node *p2p.QriNode, cli *rpc.Client) *DatasetRequests {
	return &DatasetRequests{
		node: node,
		cli:  cli,
	}
}

// NewDatasetRequestsInstance creates a DatasetRequests pointer from a qri
// instance
func NewDatasetRequestsInstance(inst *Instance) *DatasetRequests {
	return &DatasetRequests{
		node: inst.Node(),
		cli:  inst.RPC(),
		inst: inst,
	}
}

// List returns this repo's datasets
func (r *DatasetRequests) List(p *ListParams, res *[]repo.DatasetRef) error {
	if r.cli != nil {
		p.RPC = true
		return r.cli.Call("DatasetRequests.List", p, res)
	}
	ctx := context.TODO()

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

	replies, err := actions.ListDatasets(ctx, r.node, ds, p.Term, p.Limit, p.Offset, p.RPC, p.Published, p.ShowNumVersions)

	*res = replies
	return err
}

// GetParams defines parameters for looking up the body of a dataset
type GetParams struct {
	// Path to get, this will often be a dataset reference like me/dataset
	Path string

	// read from a filesystem link instead of stored version
	UseFSI       bool
	Format       string
	FormatConfig dataset.FormatConfig

	Selector string

	Limit, Offset int
	All           bool
}

// GetResult combines data with it's hashed path
type GetResult struct {
	Ref     *repo.DatasetRef `json:"ref"`
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
	ctx := context.TODO()

	ref, err := base.ToDatasetRef(p.Path, r.node.Repo, p.UseFSI)
	if err != nil {
		return err
	}

	var ds *dataset.Dataset
	if p.UseFSI {
		if ref.FSIPath == "" {
			return fsi.ErrNoLink
		}
		if ds, _, _, err = fsi.ReadDir(ref.FSIPath); err != nil {
			return fmt.Errorf("loading linked dataset: %s", err)
		}
	} else {
		ds, err = dsfs.LoadDataset(ctx, r.node.Repo.Store(), ref.Path)
		if err != nil {
			return fmt.Errorf("loading dataset: %s", err)
		}
	}

	ds.Name = ref.Name
	ds.Peername = ref.Peername
	res.Ref = ref
	res.Dataset = ds

	if err = base.OpenDataset(ctx, r.node.Repo.Filesystem(), ds); err != nil {
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

		var bufData []byte
		if p.UseFSI {
			if bufData, err = fsi.GetBody(ref.FSIPath, df, p.FormatConfig, p.Offset, p.Limit, p.All); err != nil {
				return err
			}
		} else {
			if bufData, err = actions.GetBody(r.node, ds, df, p.FormatConfig, p.Limit, p.Offset, p.All); err != nil {
				return err
			}
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
	} else if p.Selector == "rendered" && ds.Viz != nil && ds.Viz.RenderedFile() != nil {
		// `qri get rendered` loads the rendered visualization script, as a special case
		res.Bytes, err = ioutil.ReadAll(ds.Viz.RenderedFile())
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
			// Pretty defaults to true for the dataset head, unless explicitly set in the config.
			pretty := true
			if p.FormatConfig != nil {
				pvalue, ok := p.FormatConfig.Map()["pretty"].(bool)
				if ok {
					pretty = pvalue
				}
			}
			if pretty {
				res.Bytes, err = json.MarshalIndent(value, "", " ")
			} else {
				res.Bytes, err = json.Marshal(value)
			}
		case "yaml", "":
			res.Bytes, err = yaml.Marshal(value)
		default:
			return fmt.Errorf("unknown format: \"%s\"", p.Format)
		}
		return err
	}
}

// SaveParams encapsulates arguments to Save
type SaveParams struct {
	// dataset supplies params directly, all other param fields override values
	// supplied by dataset
	Dataset *dataset.Dataset

	// dataset reference string, the name to save to
	Ref string
	// commit title, defaults to a generated string based on diff
	Title string
	// commit message, defaults to blank
	Message string
	// path to body data
	BodyPath string
	// absolute path or URL to the list of dataset files or components to load
	FilePaths []string
	// secrets for transform execution
	Secrets map[string]string
	// optional writer to have transform script record standard output to
	// note: this won't work over RPC, only on local calls
	ScriptOutput io.Writer

	// load FSI-linked dataset before saving. anything provided in the Dataset
	// field and any param field will override the FSI dataset
	// read & write FSI should almost always be used in tandem, either setting
	// both to true or leaving both false
	ReadFSI bool
	// true save will write the dataset to the designated
	WriteFSI bool
	// Replace writes the entire given dataset as a new snapshot instead of
	// applying save params as augmentations to the existing history
	Replace bool
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
	// save a rendered version of the template along with the dataset
	ShouldRender bool
}

// AbsolutizePaths converts any relative path references to their absolute
// variations, safe to call on a nil instance
func (p *SaveParams) AbsolutizePaths() error {
	if p == nil {
		return nil
	}

	for i := range p.FilePaths {
		if err := qfs.AbsPath(&p.FilePaths[i]); err != nil {
			return err
		}
	}

	if err := qfs.AbsPath(&p.BodyPath); err != nil {
		return fmt.Errorf("body file: %s", err)
	}
	return nil
}

// Save adds a history entry, updating a dataset
// TODO - need to make sure users aren't forking by referencing commits other than tip
func (r *DatasetRequests) Save(p *SaveParams, res *repo.DatasetRef) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Save", p, res)
	}
	ctx := context.TODO()

	if p.Private {
		return fmt.Errorf("option to make dataset private not yet implimented, refer to https://github.com/qri-io/qri/issues/291 for updates")
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}

	ds := &dataset.Dataset{}

	if p.ReadFSI {
		err = repo.CanonicalizeDatasetRef(r.node.Repo, &ref)
		if err != nil && err != repo.ErrNoHistory {
			return err
		}
		if ref.FSIPath == "" {
			return fsi.ErrNoLink
		}

		var problems map[string]fsi.FileStat
		ds, _, problems, err = fsi.ReadDir(ref.FSIPath)
		if err != nil {
			return
		}
		if problems != nil {
			return fmt.Errorf("cannot save, dataset has errors")
		}
	}

	// add param-supplied changes
	ds.Assign(&dataset.Dataset{
		Name:     ref.Name,
		Peername: ref.Peername,
		BodyPath: p.BodyPath,
		Commit: &dataset.Commit{
			Title:   p.Title,
			Message: p.Message,
		},
	})

	if p.Dataset != nil {
		p.Dataset.Assign(ds)
		ds = p.Dataset
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
		recall, err := actions.Recall(ctx, r.node, p.Recall, ref)
		if err != nil {
			return err
		}
		recall.Assign(ds)
		ds = recall
	}

	if len(p.FilePaths) > 0 {
		// TODO (b5): handle this with a qfs.Filesystem
		dsf, err := ReadDatasetFiles(p.FilePaths...)
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

	if err = base.OpenDataset(ctx, r.node.Repo.Filesystem(), ds); err != nil {
		log.Debugf("open ds error: %s", err.Error())
		return
	}

	// TODO (b5) - this should be integrated into actions.SaveDataset
	fsiPath := ref.FSIPath

	switches := actions.SaveDatasetSwitches{
		Replace:             p.Replace,
		DryRun:              p.DryRun,
		Pin:                 true,
		ConvertFormatToPrev: p.ConvertFormatToPrev,
		Force:               p.Force,
		ShouldRender:        p.ShouldRender,
	}
	ref, err = actions.SaveDataset(ctx, r.node, ds, p.Secrets, p.ScriptOutput, switches)
	if err != nil {
		log.Debugf("create ds error: %s\n", err.Error())
		return err
	}

	// TODO (b5) - this should be integrated into actions.SaveDataset
	if fsiPath != "" {
		ref.FSIPath = fsiPath
		if err = r.node.Repo.PutRef(ref); err != nil {
			return err
		}
	}

	if p.ReturnBody {
		if err = base.InlineJSONBody(ref.Dataset); err != nil {
			return err
		}
	}

	if p.Publish {
		var publishedRef repo.DatasetRef
		err = r.SetPublishStatus(&SetPublishStatusParams{
			Ref:           ref.String(),
			PublishStatus: true,
			// UpdateRegistry:    true,
			// UpdateRegistryPin: true,
		}, &publishedRef)

		if err != nil {
			return err
		}
	}

	*res = ref

	if p.WriteFSI {
		fsi.WriteComponents(res.Dataset, ref.FSIPath)
	}
	return nil
}

// SetPublishStatusParams encapsulates parameters for setting the publication status of a dataset
type SetPublishStatusParams struct {
	Ref           string
	PublishStatus bool
	// UpdateRegistry    bool
	// UpdateRegistryPin bool
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
	ctx := context.TODO()

	if p.Current.IsEmpty() {
		return fmt.Errorf("current name is required to rename a dataset")
	}

	if err := actions.ModifyDataset(r.node, &p.Current, &p.New, true /*isRename*/); err != nil {
		return err
	}

	if err = actions.DatasetHead(ctx, r.node, &p.New); err != nil {
		log.Debug(err.Error())
		return err
	}
	*res = p.New
	return nil
}

// RemoveParams defines parameters for remove command
type RemoveParams struct {
	Ref            string
	Revision       dsref.Rev
	Unlink         bool // If true, break any FSI link
	DeleteFSIFiles bool // If true, delete tracked files from the designated FSI link
}

// RemoveResponse gives the results of a remove
type RemoveResponse struct {
	Ref             string
	NumDeleted      int
	Unlinked        bool // true if the remove unlinked an FSI-linked dataset
	DeletedFSIFiles bool // true if the remove deleted FSI-linked files
}

// Remove a dataset entirely or remove a certain number of revisions
func (r *DatasetRequests) Remove(p *RemoveParams, res *RemoveResponse) error {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Remove", p, res)
	}
	ctx := context.TODO()

	if p.Revision.Field != "ds" {
		return fmt.Errorf("can only remove whole dataset versions, not individual components")
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}

	noHistory := false
	if canonErr := repo.CanonicalizeDatasetRef(r.node.Repo, &ref); canonErr != nil && canonErr != repo.ErrNoHistory {
		return canonErr
	} else if canonErr == repo.ErrNoHistory {
		noHistory = true
	}
	res.Ref = ref.String()

	if ref.FSIPath == "" && p.Unlink {
		return fmt.Errorf("cannot unlink, dataset is not linked to a directory")
	}
	if ref.FSIPath == "" && p.DeleteFSIFiles {
		return fmt.Errorf("can't delete files, dataset is not linked to a directory")
	}

	if ref.FSIPath != "" {
		if p.DeleteFSIFiles {
			if _, err := fsi.DeleteDatasetFiles(ref.FSIPath); err != nil {
				return err
			}
			res.DeletedFSIFiles = true
		}

		// running remove on a dataset that has no history must always unlink
		if p.Unlink || noHistory {
			if err := r.inst.fsi.Unlink(ref.FSIPath, ref.AliasString()); err != nil {
				return err
			}
			res.Unlinked = true
		}
	}

	if noHistory {
		return nil
	}

	removeEntireDataset := func() error {
		// removing all revisions of a dataset must unlink it
		if ref.FSIPath != "" && !p.Unlink {
			if err := r.inst.fsi.Unlink(ref.FSIPath, ref.AliasString()); err != nil {
				return err
			}
			res.Unlinked = true
		}

		// Delete entire dataset for all generations.
		if err := actions.DeleteDataset(ctx, r.node, &ref); err != nil {
			return err
		}
		res.NumDeleted = dsref.AllGenerations

		return nil
	}

	if p.Revision.Gen == dsref.AllGenerations {
		return removeEntireDataset()
	} else if p.Revision.Gen < 1 {
		return fmt.Errorf("invalid number of revisions to delete: %d", p.Revision.Gen)
	}

	// Get the revisions that will be deleted.
	log, err := actions.DatasetLog(ctx, r.node, ref, p.Revision.Gen+1, 0)
	if err != nil {
		return err
	}

	// If deleting more revisions then exist, delete the entire dataset.
	if p.Revision.Gen >= len(log) {
		return removeEntireDataset()
	}

	// Delete the specific number of revisions.
	replace := log[p.Revision.Gen]
	if err := actions.ModifyDataset(r.node, &ref, &replace, false /*isRename*/); err != nil {
		return err
	}
	res.NumDeleted = p.Revision.Gen

	return nil
}

// AddParams encapsulates parameters to the add command
type AddParams struct {
	Ref        string
	LinkDir    string
	RemoteAddr string // remote to attempt to pull from
}

// Add adds an existing dataset to a peer's repository
func (r *DatasetRequests) Add(p *AddParams, res *repo.DatasetRef) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Add", p, res)
	}
	ctx := context.TODO()

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}

	if p.RemoteAddr == "" && r.inst != nil && r.inst.cfg.Registry != nil {
		p.RemoteAddr = r.inst.cfg.Registry.Location
	}

	if err = actions.AddDataset(ctx, r.node, r.inst.RemoteClient(), p.RemoteAddr, &ref); err != nil {
		return err
	}

	*res = ref

	if p.LinkDir != "" {
		checkoutp := &CheckoutParams{
			Ref: ref.String(),
			Dir: p.LinkDir,
		}
		m := NewFSIMethods(r.inst)
		checkoutRes := ""
		if err = m.Checkout(checkoutp, &checkoutRes); err != nil {
			return err
		}
	}

	return nil
}

// ValidateDatasetParams defines parameters for dataset
// data validation
type ValidateDatasetParams struct {
	Ref string
	// URL          string
	BodyFilename string
	Body         io.Reader
	Schema       io.Reader
}

// Validate gives a dataset of errors and issues for a given dataset
func (r *DatasetRequests) Validate(p *ValidateDatasetParams, errors *[]jsonschema.ValError) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Validate", p, errors)
	}
	ctx := context.TODO()

	// TODO: restore validating data from a URL
	// if p.URL != "" && ref.IsEmpty() && o.Schema == nil {
	//   return (lib.NewError(ErrBadArgs, "if you are validating data from a url, please include a dataset name or supply the --schema flag with a file path that Qri can validate against"))
	// }
	if p.Ref == "" && p.Body == nil && p.Schema == nil {
		return NewError(ErrBadArgs, "please provide a dataset name, or a supply the --body and --schema flags with file paths")
	}

	var body, schema qfs.File
	if p.Body != nil {
		body = qfs.NewMemfileReader(p.BodyFilename, p.Body)
	}

	if p.Schema != nil {
		schema = qfs.NewMemfileReader("schema.json", p.Schema)
	}

	var ref repo.DatasetRef
	if p.Ref != "" {
		ref, err = repo.ParseDatasetRef(p.Ref)
		if err != nil {
			return err
		}
	}

	*errors, err = actions.Validate(ctx, r.node, ref, body, schema)
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

// DAGInfoParams defines parameters for the DAGInfo method
type DAGInfoParams struct {
	RefStr, Label string
}

// DAGInfo generates a dag.Info for a dataset path. If a label is given, DAGInfo will generate a sub-dag.Info at that label.
func (r *DatasetRequests) DAGInfo(s *DAGInfoParams, i *dag.Info) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.DAGInfo", s, i)
	}

	ref, err := repo.ParseDatasetRef(s.RefStr)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(r.node.Repo, &ref); err != nil {
		return
	}

	var info *dag.Info
	info, err = actions.NewDAGInfo(r.node, ref.Path, s.Label)
	if err != nil {
		return
	}
	*i = *info
	return
}
