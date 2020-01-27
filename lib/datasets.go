package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/qri-io/dag"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/base/fill"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
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

// List gets the reflist for either the local repo or a peer
func (r *DatasetRequests) List(p *ListParams, res *[]repo.DatasetRef) error {
	if r.cli != nil {
		p.RPC = true
		return r.cli.Call("DatasetRequests.List", p, res)
	}
	ctx := context.TODO()

	// ensure valid limit value
	if p.Limit <= 0 {
		p.Limit = 25
	}
	// ensure valid offset value
	if p.Offset < 0 {
		p.Offset = 0
	}

	// TODO (b5) - this logic around weather we're listing locally or
	// a remote peer needs cleanup
	ref := &repo.DatasetRef{
		Peername:  p.Peername,
		ProfileID: p.ProfileID,
	}
	if err := repo.CanonicalizeProfile(r.node.Repo, ref); err != nil {
		return fmt.Errorf("error canonicalizing peer: %s", err.Error())
	}

	pro, err := r.node.Repo.Profile()
	if err != nil {
		return err
	}

	var refs []repo.DatasetRef
	if p.ViaDscache {
		c, err := dscache.BuildDscacheFromLogbookAndProfilesAndDsref(r.node.Repo)
		if err != nil {
			return err
		}
		refs, err = c.ListRefs()
		// TODO(dlong): Set dataset field for each reference, which will end up displaying
		// title and basic structure stats (such as size, entries, errors).
	} else if ref.Peername == "" || pro.Peername == ref.Peername {
		refs, err = base.ListDatasets(ctx, r.node.Repo, p.Term, p.Limit, p.Offset, p.RPC, p.Published, p.ShowNumVersions)
	} else {

		refs, err = r.inst.RemoteClient().ListDatasets(ctx, ref, p.Term, p.Offset, p.Limit)
	}
	if err != nil {
		return err
	}

	if p.EnsureFSIExists {
		// For each reference with a linked fsi working directory, check that the folder exists
		// and has a .qri-ref file. If it's missing, remove the link from the centralized repo.
		// Doing this every list operation is a bit inefficient, so the behavior is opt-in.
		for _, ref := range refs {
			if ref.FSIPath != "" {
				target := filepath.Join(ref.FSIPath, fsi.QriRefFilename)
				_, err := os.Stat(target)
				if os.IsNotExist(err) {
					ref.FSIPath = ""
					if ref.Path == "" {
						if err = r.node.Repo.DeleteRef(ref); err != nil {
							log.Debugf("cannot delete ref for %q, err: %s", ref, err)
						}
						continue
					}
					if err = r.node.Repo.PutRef(ref); err != nil {
						log.Debugf("cannot put ref for %q, err: %s", ref, err)
					}
				}
			}
		}
	}

	*res = refs

	// TODO (b5) - for now we're removing schemas b/c they don't serialize properly over RPC
	// update 2019-10-21 - this probably isn't true anymore. should test & remove
	if p.RPC {
		for _, rep := range *res {
			if rep.Dataset != nil && rep.Dataset.Structure != nil {
				rep.Dataset.Structure.Schema = nil
			}
		}
	}

	return err
}

// ListRawRefs gets the list of raw references as string
func (r *DatasetRequests) ListRawRefs(p *ListParams, text *string) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.ListRawRefs", p, text)
	}
	if p.ViaDscache {
		// NOTE: Useful for debugging. Only outputting to local terminal for now.
		c, err := dscache.BuildDscacheFromLogbookAndProfilesAndDsref(r.node.Repo)
		if err != nil {
			return err
		}
		c.Dump()
		return nil
	}
	*text, err = base.RawDatasetRefs(context.TODO(), r.node.Repo)
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
// then res.Bytes is loaded with the body. If the selector is "stats", then res.Bytes is loaded
// with the generated stats.
func (r *DatasetRequests) Get(p *GetParams, res *GetResult) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Get", p, res)
	}
	ctx := context.TODO()

	ref, err := base.ToDatasetRef(p.Path, r.node.Repo, p.UseFSI)
	if err != nil {
		log.Debugf("Get dataset, base.ToDatasetRef %q failed, error: %s", p.Path, err)
		return err
	}

	var ds *dataset.Dataset
	if p.UseFSI {
		if ref.FSIPath == "" {
			log.Debugf("Get dataset, p.Path %q, ref %q failed, ref.FSIPath is empty", p.Path, ref)
			return fsi.ErrNoLink
		}
		if ds, err = fsi.ReadDir(ref.FSIPath); err != nil {
			log.Debugf("Get dataset, fsi.ReadDir %q failed, error: %s", ref.FSIPath, err)
			return fmt.Errorf("loading linked dataset: %s", err)
		}
	} else {
		ds, err = dsfs.LoadDataset(ctx, r.node.Repo.Store(), ref.Path)
		if err != nil {
			log.Debugf("Get dataset, dsfs.LoadDataset %q failed, error: %s", ref, err)
			return fmt.Errorf("loading dataset: %s", err)
		}
	}

	ds.Name = ref.Name
	ds.Peername = ref.Peername
	res.Ref = ref
	res.Dataset = ds

	if err = base.OpenDataset(ctx, r.node.Repo.Filesystem(), ds); err != nil {
		log.Debugf("Get dataset, base.OpenDataset failed, error: %s", err)
		return err
	}

	if p.Selector == "body" {
		// `qri get body` loads the body
		if !p.All && (p.Limit < 0 || p.Offset < 0) {
			return fmt.Errorf("invalid limit / offset settings")
		}
		df, err := dataset.ParseDataFormatString(p.Format)
		if err != nil {
			log.Debugf("Get dataset, ParseDataFormatString %q failed, error: %s", p.Format, err)
			return err
		}

		var bufData []byte
		if p.UseFSI {
			if bufData, err = fsi.GetBody(ref.FSIPath, df, p.FormatConfig, p.Offset, p.Limit, p.All); err != nil {
				log.Debugf("Get dataset, fsi.GetBody %q failed, error: %s", ref.FSIPath, err)
				return err
			}
		} else {
			if bufData, err = base.ReadBody(ds, df, p.FormatConfig, p.Limit, p.Offset, p.All); err != nil {
				log.Debugf("Get dataset, base.ReadBody %q failed, error: %s", ds, err)
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
	} else if p.Selector == "readme.script" && ds.Readme != nil && ds.Readme.ScriptFile() != nil {
		// `qri get readme.script` loads the readme source, as a special case
		res.Bytes, err = ioutil.ReadAll(ds.Readme.ScriptFile())
		return err
	} else if p.Selector == "viz.script" && ds.Viz != nil && ds.Viz.ScriptFile() != nil {
		// `qri get viz.script` loads the visualization script, as a special case
		res.Bytes, err = ioutil.ReadAll(ds.Viz.ScriptFile())
		return err
	} else if p.Selector == "rendered" && ds.Viz != nil && ds.Viz.RenderedFile() != nil {
		// `qri get rendered` loads the rendered visualization script, as a special case
		res.Bytes, err = ioutil.ReadAll(ds.Viz.RenderedFile())
		return err
	} else if p.Selector == "stats" {
		statsParams := &StatsParams{
			Dataset: res.Dataset,
		}
		statsRes := &StatsResponse{}
		if err := r.Stats(statsParams, statsRes); err != nil {
			return err
		}
		res.Bytes = statsRes.StatsBytes
		return
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
	// new dataset only, don't create a commit on an existing dataset, name will be unused
	NewName bool
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
		return fmt.Errorf("option to make dataset private not yet implemented, refer to https://github.com/qri-io/qri/issues/291 for updates")
	}

	// From cmd/, an empty reference becomes "me/", but from api/, it becomes "" (empty string).
	// We must allow empty references for the case when the --new flag is being used, since it
	// can be used to generate a name from a bodypath.
	// TODO(dlong): Fix me! Check for these cases, return reasonable errors, test at lib/ level.
	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil && err != repo.ErrEmptyRef {
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

		ds, err = fsi.ReadDir(ref.FSIPath)
		if err != nil {
			return
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
		recall, err := base.Recall(ctx, r.node.Repo, p.Recall, ref)
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

	if p.BodyPath == "" && ds.Name == "" {
		return fmt.Errorf("name or bodypath is required")
	}
	if !p.Force &&
		ds.BodyPath == "" &&
		ds.Body == nil &&
		ds.BodyBytes == nil &&
		ds.Structure == nil &&
		ds.Meta == nil &&
		ds.Readme == nil &&
		ds.Viz == nil &&
		ds.Transform == nil {
		return fmt.Errorf("no changes to save")
	}

	if err = base.OpenDataset(ctx, r.node.Repo.Filesystem(), ds); err != nil {
		log.Debugf("open ds error: %s", err.Error())
		return
	}

	// TODO (b5) - this should be integrated into base.SaveDataset
	fsiPath := ref.FSIPath

	switches := base.SaveDatasetSwitches{
		Replace:             p.Replace,
		DryRun:              p.DryRun,
		Pin:                 true,
		ConvertFormatToPrev: p.ConvertFormatToPrev,
		Force:               p.Force,
		ShouldRender:        p.ShouldRender,
		NewName:             p.NewName,
	}
	ref, err = base.SaveDataset(ctx, r.node.Repo, r.node.LocalStreams, ds, p.Secrets, p.ScriptOutput, switches)
	if err != nil {
		log.Debugf("create ds error: %s\n", err.Error())
		return err
	}

	// TODO (b5) - this should be integrated into base.SaveDataset
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
		// Need to pass filesystem here so that we can read the README component and write it
		// properly back to disk.
		fsi.WriteComponents(res.Dataset, ref.FSIPath, r.inst.node.Repo.Filesystem())
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
	if err = base.SetPublishStatus(r.node.Repo, &ref, ref.Published); err != nil {
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

	// Update the reference stored in the repo
	if err := base.ModifyDatasetRef(ctx, r.node.Repo, &p.Current, &p.New, true /*isRename*/); err != nil {
		return err
	}

	// If the dataset is linked to a working directory, update the ref
	if p.New.FSIPath != "" {
		if err = r.inst.fsi.ModifyLinkReference(p.New.FSIPath, p.New.String()); err != nil {
			return err
		}
	}

	if err = base.ReadDataset(ctx, r.node.Repo, &p.New); err != nil && err != repo.ErrNoHistory {
		log.Debug(err.Error())
		return err
	}
	*res = p.New
	return nil
}

// RemoveParams defines parameters for remove command
type RemoveParams struct {
	Ref       string
	Revision  dsref.Rev
	KeepFiles bool
	Force     bool
}

// RemoveResponse gives the results of a remove
type RemoveResponse struct {
	Ref        string
	NumDeleted int
	Unlinked   bool
}

// ErrCantRemoveDirectoryDirty is returned when a directory is dirty so the files cant' be removed
var ErrCantRemoveDirectoryDirty = fmt.Errorf("cannot remove files while working directory is dirty")

// Remove a dataset entirely or remove a certain number of revisions
func (r *DatasetRequests) Remove(p *RemoveParams, res *RemoveResponse) error {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Remove", p, res)
	}
	ctx := context.TODO()

	if p.Revision.Gen == 0 {
		return fmt.Errorf("invalid number of revisions to delete: 0")
	}

	if p.Revision.Field != "ds" {
		return fmt.Errorf("can only remove whole dataset versions, not individual components")
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}

	if canonErr := repo.CanonicalizeDatasetRef(r.node.Repo, &ref); canonErr != nil && canonErr != repo.ErrNoHistory {
		return canonErr
	}
	res.Ref = ref.String()

	if ref.FSIPath != "" {
		// Dataset is linked in a working directory.
		if !(p.KeepFiles || p.Force) {
			// Make sure that status is clean, otherwise, refuse to remove any revisions.
			// Setting either --keep-files or --force will skip this check.
			wdErr := r.inst.fsi.IsWorkingDirectoryClean(ctx, ref.FSIPath)
			if wdErr != nil {
				if wdErr == fsi.ErrWorkingDirectoryDirty {
					return ErrCantRemoveDirectoryDirty
				}
				if strings.Contains(wdErr.Error(), "not a linked directory") {
					// If the working directory has been removed (or renamed), could not get the
					// status. However, don't let this stop the remove operation, since the files
					// are already gone, and therefore won't be removed.
					log.Debugf("could not get status for %s, maybe removed or renamed", ref.FSIPath)
					wdErr = nil
				} else {
					return wdErr
				}
			}
		}
	} else if p.KeepFiles {
		// If dataset is not linked in a working directory, --keep-files can't be used.
		return fmt.Errorf("dataset is not linked to filesystem, cannot use keep-files")
	}

	// Get the revisions that will be deleted.
	history, err := base.DatasetLog(ctx, r.node.Repo, ref, p.Revision.Gen+1, 0, false)
	if err != nil {
		if err == repo.ErrNoHistory {
			p.Revision.Gen = dsref.AllGenerations
		} else {
			return err
		}
	}

	if p.Revision.Gen == dsref.AllGenerations || p.Revision.Gen >= len(history) {
		// removing all revisions of a dataset must unlink it
		if ref.FSIPath != "" {
			if err := r.inst.fsi.Unlink(ref.FSIPath, ref.AliasString()); err != nil {
				return err
			}
			res.Unlinked = true
		}

		// If the dataset has no history (such as running `qri init` without `qri save`), then
		// the ref has no path. Can't call RemoveNVersionsFromStore without a path, but don't
		// need to call it anyway. Skip it.
		if len(history) > 0 {
			// Delete entire dataset for all generations.
			if _, err := base.RemoveNVersionsFromStore(ctx, r.inst.Repo(), &ref, -1); err != nil {
				return err
			}
		}
		// Write the deletion to the logbook.
		book := r.inst.Repo().Logbook()
		if err := book.WriteDatasetDelete(ctx, repo.ConvertToDsref(ref)); err != nil {
			// If the logbook is missing, it's not an error worth stopping for, since we're
			// deleting the dataset anyway. This can happen from adding a foreign dataset.
			if err != oplog.ErrNotFound {
				return err
			}
		}
		// remove the ref from the ref store
		if err := r.inst.Repo().DeleteRef(ref); err != nil {
			return err
		}
		res.NumDeleted = dsref.AllGenerations

		if ref.FSIPath != "" && !p.KeepFiles {
			// Remove all files
			fsi.DeleteComponentFiles(ref.FSIPath)
			// Delete the directory
			err = os.Remove(ref.FSIPath)
			if err != nil {
				if strings.Contains(err.Error(), "no such file or directory") {
					// If the working directory has already been removed (or renamed), it is
					// not an error that this remove operation fails, since we were trying to
					// remove them anyway.
					log.Debugf("could not remove %s, maybe already removed or renamed", ref.FSIPath)
					err = nil
				} else {
					return err
				}
			}
		}
	} else {
		// Delete the specific number of revisions.
		dsr := history[p.Revision.Gen]
		replace := &repo.DatasetRef{
			Peername:  dsr.Ref.Username,
			Name:      dsr.Ref.Name,
			ProfileID: ref.ProfileID, // TODO (b5) - this is a cheat for now
			Path:      dsr.Ref.Path,
			Published: dsr.Published,
		}
		err = base.ModifyDatasetRef(ctx, r.node.Repo, &ref, replace, false /*isRename*/)
		if err != nil {
			return err
		}
		head, err := base.RemoveNVersionsFromStore(ctx, r.inst.Repo(), &ref, p.Revision.Gen)
		if err != nil {
			return err
		}
		res.NumDeleted = p.Revision.Gen

		if ref.FSIPath != "" && !p.KeepFiles {
			// Load dataset version that is at head after newer versions are removed
			ds, err := dsfs.LoadDataset(ctx, r.inst.Repo().Store(), head.Path)
			if err != nil {
				return err
			}
			ds.Name = head.Name
			ds.Peername = head.Peername
			if err = base.OpenDataset(ctx, r.inst.Repo().Filesystem(), ds); err != nil {
				return err
			}

			// TODO(dlong): Add a method to FSI called ProjectOntoDirectory, use it here
			// and also for Restore() in lib/fsi.go and also maybe WriteComponents in fsi/mapping.go

			// Delete the old files
			err = fsi.DeleteComponentFiles(ref.FSIPath)
			if err != nil {
				log.Debug("deleting component files: %s", err)
			}

			// Update the files in the working directory
			fsi.WriteComponents(ds, ref.FSIPath, r.inst.node.Repo.Filesystem())
		}
	}
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
	if err = qfs.AbsPath(&p.LinkDir); err != nil {
		return
	}

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

	// TODO (b5) - we're early in log syncronization days. This is going to fail a bunch
	// while we work to upgrade the stack. Long term we may want to consider a mechanism
	// for allowing partial completion where only one of logs or dataset pulling works
	// by doing both in parallel and reporting issues on both
	if pullLogsErr := r.inst.RemoteClient().PullLogs(ctx, repo.ConvertToDsref(ref), p.RemoteAddr); pullLogsErr != nil {
		log.Errorf("pulling logs: %s", pullLogsErr)
	}

	if err = r.inst.RemoteClient().AddDataset(ctx, &ref, p.RemoteAddr); err != nil {
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
	BodyFilename      string
	SchemaFilename    string
	StructureFilename string
	UseFSI            bool
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

	// Schema can come from either schema.json or structure.json, or the dataset itself.
	// schemaFlagType determines which of these three contains the schema.
	schemaFlagType := ""
	schemaFilename := ""
	if p.SchemaFilename != "" && p.StructureFilename != "" {
		return NewError(ErrBadArgs, "cannot provide both --schema and --structure flags")
	} else if p.SchemaFilename != "" {
		schemaFlagType = "schema"
		schemaFilename = p.SchemaFilename
	} else if p.StructureFilename != "" {
		schemaFlagType = "structure"
		schemaFilename = p.StructureFilename
	}

	if p.Ref == "" && (p.BodyFilename == "" || schemaFlagType == "") {
		return NewError(ErrBadArgs, "please provide a dataset name, or a supply the --body and --schema or --structure flags")
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil && err != repo.ErrEmptyRef {
		return err
	}
	err = repo.CanonicalizeDatasetRef(r.node.Repo, &ref)
	if err != nil && err != repo.ErrEmptyRef {
		if err == repo.ErrNotFound {
			return fmt.Errorf("cannot find dataset: %s", ref)
		}
		return err
	}

	var ds *dataset.Dataset

	// TODO(dlong): This pattern has shown up many places, such as lib.Get.
	// Should probably combine into a utility function.

	if p.Ref != "" {
		if p.UseFSI {
			if ref.FSIPath == "" {
				return fsi.ErrNoLink
			}
			if ds, err = fsi.ReadDir(ref.FSIPath); err != nil {
				return fmt.Errorf("loading linked dataset: %s", err)
			}
		} else {
			if ds, err = dsfs.LoadDataset(ctx, r.node.Repo.Store(), ref.Path); err != nil {
				return fmt.Errorf("loading dataset: %s", err)
			}
		}
		if err = base.OpenDataset(ctx, r.node.Repo.Filesystem(), ds); err != nil {
			return err
		}
	}

	var body qfs.File
	if p.BodyFilename == "" {
		body = ds.BodyFile()
	} else {
		// Body is set to the provided filename if given
		fs := localfs.NewFS()
		body, err = fs.Get(context.Background(), p.BodyFilename)
		if err != nil {
			return fmt.Errorf("error opening body file: %s", p.BodyFilename)
		}
	}

	var st *dataset.Structure
	// Schema is set to the provided filename if given, otherwise the dataset's schema
	if schemaFlagType == "" {
		st = ds.Structure
	} else {
		data, err := ioutil.ReadFile(schemaFilename)
		if err != nil {
			return fmt.Errorf("error opening schema file: %s", p.SchemaFilename)
		}
		var fileContent map[string]interface{}
		err = json.Unmarshal(data, &fileContent)
		if err != nil {
			return err
		}
		if schemaFlagType == "schema" {
			// If dataset ref was provided, get format from the structure. Otherwise, assume the
			// format by looking at the body file's extension.
			var bodyFormat string
			if ds != nil && ds.Structure != nil {
				bodyFormat = ds.Structure.Format
			} else {
				bodyFormat = filepath.Ext(p.BodyFilename)
				if strings.HasSuffix(bodyFormat, ".") {
					bodyFormat = bodyFormat[1:]
				}
			}
			st = &dataset.Structure{
				Format: bodyFormat,
				Schema: fileContent,
			}
		} else {
			// schemaFlagType == "structure". Deserialize the provided file into a structure.
			st = &dataset.Structure{}
			err = fill.Struct(fileContent, st)
			if err != nil {
				return err
			}
			// TODO(dlong): What happens if body file extension does not match st.Format?
		}
	}

	*errors, err = base.Validate(ctx, r.node.Repo, body, st)
	return
}

// Manifest generates a manifest for a dataset path
func (r *DatasetRequests) Manifest(refstr *string, m *dag.Manifest) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Manifest", refstr, m)
	}
	ctx := context.TODO()

	ref, err := repo.ParseDatasetRef(*refstr)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(r.node.Repo, &ref); err != nil {
		return
	}

	var mf *dag.Manifest
	mf, err = r.node.NewManifest(ctx, ref.Path)
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
	ctx := context.TODO()

	var mf *dag.Manifest
	mf, err = r.node.MissingManifest(ctx, a)
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
	ctx := context.TODO()

	ref, err := repo.ParseDatasetRef(s.RefStr)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(r.node.Repo, &ref); err != nil {
		return
	}

	var info *dag.Info
	info, err = r.node.NewDAGInfo(ctx, ref.Path, s.Label)
	if err != nil {
		return
	}
	*i = *info
	return
}

// StatsParams defines the params for a Stats request
type StatsParams struct {
	// string representation of a dataset reference
	Ref string
	// if we get a Dataset from the params, then we do not have to
	// attempt to open a dataset from the reference
	Dataset *dataset.Dataset
}

// StatsResponse defines the response for a Stats request
type StatsResponse struct {
	StatsBytes []byte
}

// Stats generates stats for a dataset
func (r *DatasetRequests) Stats(p *StatsParams, res *StatsResponse) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Stats", p, res)
	}
	ctx := context.TODO()
	if p.Dataset == nil {
		ref := &repo.DatasetRef{}
		ref, err = base.ToDatasetRef(p.Ref, r.node.Repo, false)
		if err != nil {
			return err
		}
		p.Dataset, err = dsfs.LoadDataset(ctx, r.node.Repo.Store(), ref.Path)
		if err != nil {
			return fmt.Errorf("loading dataset: %s", err)
		}

		if err = base.OpenDataset(ctx, r.node.Repo.Filesystem(), p.Dataset); err != nil {
			return
		}
	}
	if p.Dataset.Structure == nil || p.Dataset.Structure.IsEmpty() {
		p.Dataset.Structure = &dataset.Structure{}
		p.Dataset.Structure.Format = filepath.Ext(p.Dataset.BodyFile().FileName())
		p.Dataset.Structure.Schema, _, err = detect.Schema(p.Dataset.Structure, p.Dataset.BodyFile())
		if err != nil {
			return err
		}
		// TODO (ramfox): this feels gross, but since we consume the reader when
		// detecting the schema, we need to open up the file again, since we don't
		// have the option to seek back to the front
		if err = p.Dataset.OpenBodyFile(ctx, r.node.Repo.Filesystem()); err != nil {
			return err
		}
	}
	reader, err := r.inst.stats.JSON(ctx, p.Dataset)
	if err != nil {
		return err
	}
	res.StatsBytes, err = ioutil.ReadAll(reader)
	return err
}
