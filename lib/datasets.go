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
	"github.com/qri-io/qri/dscache/build"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
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
func (r *DatasetRequests) List(p *ListParams, res *[]dsref.VersionInfo) error {
	if r.cli != nil {
		p.RPC = true
		return checkRPCError(r.cli.Call("DatasetRequests.List", p, res))
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
	ref := &reporef.DatasetRef{
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

	var refs []reporef.DatasetRef
	if p.UseDscache {
		c := r.node.Repo.Dscache()
		if c.IsEmpty() {
			log.Infof("building dscache from repo's logbook, profile, and dsref")
			built, err := build.DscacheFromRepo(ctx, r.node.Repo)
			if err != nil {
				return err
			}
			err = c.Assign(built)
			if err != nil {
				log.Error(err)
			}
		}
		refs, err = c.ListRefs()
		if err != nil {
			return err
		}
		// Filter references so that only with a matching name are returned
		if p.Term != "" {
			matched := make([]reporef.DatasetRef, len(refs))
			count := 0
			for _, ref := range refs {
				if strings.Contains(ref.AliasString(), p.Term) {
					matched[count] = ref
					count++
				}
			}
			refs = matched[:count]
		}
		// Filter references by skipping to the correct offset
		if p.Offset > len(refs) {
			refs = []reporef.DatasetRef{}
		} else {
			refs = refs[p.Offset:]
		}
		// Filter references by limiting how many are returned
		if p.Limit < len(refs) {
			refs = refs[:p.Limit]
		}
		// TODO(dlong): Filtered by p.Published flag
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

	// Convert old style DatasetRef list to VersionInfo list.
	// TODO(dlong): Remove this and convert lower-level functions to return []VersionInfo.
	infos := make([]dsref.VersionInfo, len(refs))
	for i, r := range refs {
		infos[i] = reporef.ConvertToVersionInfo(&r)
	}
	*res = infos

	return err
}

// ListRawRefs gets the list of raw references as string
func (r *DatasetRequests) ListRawRefs(p *ListParams, text *string) (err error) {
	if r.cli != nil {
		return checkRPCError(r.cli.Call("DatasetRequests.ListRawRefs", p, text))
	}
	ctx := context.TODO()
	if p.UseDscache {
		c := r.node.Repo.Dscache()
		if c == nil || c.IsEmpty() {
			return fmt.Errorf("repo: dscache not found")
		}
		*text = c.VerboseString(true)
		return nil
	}
	*text, err = base.RawDatasetRefs(ctx, r.node.Repo)
	return err
}

// GetParams defines parameters for looking up the body of a dataset
type GetParams struct {
	// Ref to get, for example a dataset reference like me/dataset
	Ref string

	// read from a filesystem link instead of stored version
	Format       string
	FormatConfig dataset.FormatConfig

	Selector string

	Limit, Offset int
	All           bool
}

// GetResult combines data with it's hashed path
type GetResult struct {
	Ref     *reporef.DatasetRef `json:"ref"`
	Dataset *dataset.Dataset    `json:"data"`
	Bytes   []byte              `json:"bytes"`
}

// Get retrieves datasets and components for a given reference. If p.Ref is provided, it is
// used to load the dataset, otherwise p.Ref is parsed to create a reference. The
// dataset will be loaded from the local repo if available, or by asking peers for it.
// Using p.Selector will control what components are returned in res.Bytes. The default,
// a blank selector, will also fill the entire dataset at res.Data. If the selector is "body"
// then res.Bytes is loaded with the body. If the selector is "stats", then res.Bytes is loaded
// with the generated stats.
func (r *DatasetRequests) Get(p *GetParams, res *GetResult) (err error) {
	if r.cli != nil {
		return checkRPCError(r.cli.Call("DatasetRequests.Get", p, res))
	}
	ctx := context.TODO()

	// Check if the dataset ref uses bad-case characters, show a warning.
	dr, err := dsref.Parse(p.Ref)
	if err == dsref.ErrBadCaseName {
		log.Error(dsref.ErrBadCaseShouldRename)
	}

	ref, err := base.ToDatasetRef(p.Ref, r.node.Repo, true)
	if err != nil {
		log.Debugf("Get dataset, base.ToDatasetRef %q failed, error: %s", p.Ref, err)
		return err
	}

	var ds *dataset.Dataset
	if dr.Path == "" && ref.FSIPath != "" {
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
		if dr.Path == "" && ref.FSIPath != "" {
			// TODO(dustmop): Need to handle the special case where an FSI directory has a body
			// but no structure, which should infer a schema in order to read the body. Once that
			// works we can remove the fsi.GetBody call and just use base.ReadBody.
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
	} else if scriptFile, ok := scriptFileSelection(ds, p.Selector); ok {
		// Fields that have qfs.File types should be read and returned
		res.Bytes, err = ioutil.ReadAll(scriptFile)
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

func scriptFileSelection(ds *dataset.Dataset, selector string) (qfs.File, bool) {
	parts := strings.Split(selector, ".")
	if len(parts) != 2 {
		return nil, false
	}
	if parts[1] != "script" {
		return nil, false
	}
	if parts[0] == "transform" && ds.Transform != nil && ds.Transform.ScriptFile() != nil {
		return ds.Transform.ScriptFile(), true
	} else if parts[0] == "readme" && ds.Readme != nil && ds.Readme.ScriptFile() != nil {
		return ds.Readme.ScriptFile(), true
	} else if parts[0] == "viz" && ds.Viz != nil && ds.Viz.ScriptFile() != nil {
		return ds.Viz.ScriptFile(), true
	} else if parts[0] == "rendered" && ds.Viz != nil && ds.Viz.RenderedFile() != nil {
		return ds.Viz.RenderedFile(), true
	}
	return nil, false
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
	// whether to create a new dscache if none exists
	UseDscache bool
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
func (r *DatasetRequests) Save(p *SaveParams, res *reporef.DatasetRef) (err error) {
	if r.cli != nil {
		p.ScriptOutput = nil
		return checkRPCError(r.cli.Call("DatasetRequests.Save", p, res))
	}
	ctx := context.TODO()

	if p.Private {
		return fmt.Errorf("option to make dataset private not yet implemented, refer to https://github.com/qri-io/qri/issues/291 for updates")
	}

	ref, err := dsref.ParseHumanFriendly(p.Ref)
	if err == dsref.ErrBadCaseName {
		// If dataset name is using bad-case characters, and is not yet in use, fail with error.
		if !r.nameIsInUse(ref) {
			return err
		}
		// If dataset name already exists, just log a warning and then continue.
		log.Error(dsref.ErrBadCaseShouldRename)
	} else if err == dsref.ErrEmptyRef {
		// Okay if reference is empty. Later code will try to infer the name from other parameters.
	} else if err != nil {
		// If some other error happened, return that error.
		return err
	}

	// Validate that username is our own, it's not valid to try to save a dataset with someone
	// else's username. Without this check, base will replace the username with our own regardless,
	// it's better to have an error to display, rather than silently ignore it.
	pro, err := r.node.Repo.Profile()
	if err != nil {
		return err
	}
	if ref.Username != "" && ref.Username != "me" && ref.Username != pro.Peername {
		return fmt.Errorf("cannot save using a different username than \"%s\"", pro.Peername)
	}

	// Parsed human-friendly dsref can only have username and name.
	datasetRef := reporef.DatasetRef{
		Peername: ref.Username,
		Name:     ref.Name,
	}

	ds := &dataset.Dataset{}

	// Check if the dataset has an FSIPath, which requires a different save codepath.
	err = repo.CanonicalizeDatasetRef(r.node.Repo, &datasetRef)
	// Ignore errors that happen when saving a new dataset for the first time
	if err == repo.ErrNotFound || err == repo.ErrEmptyRef {
		// do nothing
	} else if err == nil || err == repo.ErrNoHistory {
		// When saving in an FSI directory, the ref should exist (due to `qri init`), and we
		// need to load the previous version from the working directory.
		if datasetRef.FSIPath != "" {
			ds, err = fsi.ReadDir(datasetRef.FSIPath)
			if err != nil {
				return err
			}
		}
	} else {
		return err
	}

	// add param-supplied changes
	ds.Assign(&dataset.Dataset{
		Name:     datasetRef.Name,
		Peername: datasetRef.Peername,
		BodyPath: p.BodyPath,
		Commit: &dataset.Commit{
			Title:   p.Title,
			Message: p.Message,
		},
	})

	// TODO(dustmop): A hack! Before, we were only calling CanonializeDatasetRef when saving to
	// an FSI linked directory, so the Username "me" was not being replaced by the true username.
	// Somehow this value is getting into IPFS, and a lot of tests break due to ipfs hashes
	// changing. Fix this in a follow-up change, and verify that usernames are never saved into
	// filestore.
	if ref.Username == "me" {
		ds.Peername = "me"
	}

	if p.Dataset != nil {
		p.Dataset.Assign(ds)
		ds = p.Dataset
	}

	if p.Recall != "" {
		datasetRef := reporef.DatasetRef{
			Peername: ds.Peername,
			Name:     ds.Name,
			// TODO - fix, but really this should be fine for a while because
			// ProfileID is required to be local when saving
			// ProfileID: ds.ProfileID,
			Path: ds.Path,
		}
		recall, err := base.Recall(ctx, r.node.Repo, p.Recall, datasetRef)
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

	// If the dscache doesn't exist yet, it will only be created if the appropriate flag enables it.
	if p.UseDscache {
		c := r.node.Repo.Dscache()
		c.CreateNewEnabled = true
	}

	// TODO (b5) - this should be integrated into base.SaveDataset
	fsiPath := datasetRef.FSIPath

	switches := base.SaveDatasetSwitches{
		Replace:             p.Replace,
		DryRun:              p.DryRun,
		Pin:                 true,
		ConvertFormatToPrev: p.ConvertFormatToPrev,
		Force:               p.Force,
		ShouldRender:        p.ShouldRender,
		NewName:             p.NewName,
	}
	datasetRef, err = base.SaveDataset(ctx, r.node.Repo, r.node.LocalStreams, ds, p.Secrets, p.ScriptOutput, switches)
	if err != nil {
		log.Debugf("create ds error: %s\n", err.Error())
		return err
	}

	// TODO (b5) - this should be integrated into base.SaveDataset
	if fsiPath != "" {
		datasetRef.FSIPath = fsiPath
		if err = r.node.Repo.PutRef(datasetRef); err != nil {
			return err
		}
	}

	if p.ReturnBody {
		if err = base.InlineJSONBody(datasetRef.Dataset); err != nil {
			return err
		}
	}

	if p.Publish {
		var publishedRef reporef.DatasetRef
		err = r.SetPublishStatus(&SetPublishStatusParams{
			Ref:           datasetRef.String(),
			PublishStatus: true,
			// UpdateRegistry:    true,
			// UpdateRegistryPin: true,
		}, &publishedRef)

		if err != nil {
			return err
		}
	}

	*res = datasetRef

	if fsiPath != "" {
		// Need to pass filesystem here so that we can read the README component and write it
		// properly back to disk.
		fsi.WriteComponents(res.Dataset, datasetRef.FSIPath, r.inst.node.Repo.Filesystem())
	}
	return nil
}

// This is somewhat of a hack, we shouldn't need to lookup anything about the dataset reference
// before running Save. However, we need to check for now until we solve the problem of
// dataset names existing with bad-case characters.
// See this issue: https://github.com/qri-io/qri/issues/1132
func (r *DatasetRequests) nameIsInUse(ref dsref.Ref) bool {
	param := GetParams{
		Ref: ref.Alias(),
	}
	res := GetResult{}
	err := r.Get(&param, &res)
	if err == repo.ErrNotFound {
		return false
	}
	if err != nil {
		// TODO(dustmop): Unsure if this is correct. If `Get` hits some other error, we aren't
		// sure if the dataset name is in use. Log the error and assume the dataset does in fact
		// exist.
		log.Error(err)
	}
	return true
}

// SetPublishStatusParams encapsulates parameters for setting the publication status of a dataset
type SetPublishStatusParams struct {
	Ref           string
	PublishStatus bool
	// UpdateRegistry    bool
	// UpdateRegistryPin bool
}

// SetPublishStatus updates the publicity of a reference in the peer's namespace
func (r *DatasetRequests) SetPublishStatus(p *SetPublishStatusParams, publishedRef *reporef.DatasetRef) (err error) {
	if r.cli != nil {
		return checkRPCError(r.cli.Call("DatasetRequests.SetPublishStatus", p, publishedRef))
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
	Current, Next dsref.Ref
}

// Rename changes a user's given name for a dataset
func (r *DatasetRequests) Rename(p *RenameParams, res *dsref.VersionInfo) (err error) {
	if r.cli != nil {
		return checkRPCError(r.cli.Call("DatasetRequests.Rename", p, res))
	}
	ctx := context.TODO()

	if p.Current.IsEmpty() {
		return fmt.Errorf("current name is required to rename a dataset")
	}

	// Update the reference stored in the repo
	info, err := base.ModifyDatasetRef(ctx, r.node.Repo, p.Current, p.Next)
	if err != nil {
		return err
	}

	// If the dataset is linked to a working directory, update the ref
	if info.FSIPath != "" {
		if err = r.inst.fsi.ModifyLinkReference(info.FSIPath, info.Alias()); err != nil {
			return err
		}
	}

	pid, err := profile.IDB58Decode(info.ProfileID)
	if err != nil {
		pid = ""
	}

	readRef := reporef.DatasetRef{
		Peername:  info.Username,
		ProfileID: pid,
		Name:      info.Name,
		Path:      info.Path,
	}

	if err = base.ReadDataset(ctx, r.node.Repo, &readRef); err != nil && err != repo.ErrNoHistory {
		log.Debug(err.Error())
		return err
	}
	*res = *info
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
	Message    string
	Unlinked   bool
}

// ErrCantRemoveDirectoryDirty is returned when a directory is dirty so the files cant' be removed
var ErrCantRemoveDirectoryDirty = fmt.Errorf("cannot remove files while working directory is dirty")

// Remove a dataset entirely or remove a certain number of revisions
func (r *DatasetRequests) Remove(p *RemoveParams, res *RemoveResponse) error {
	if r.cli != nil {
		return checkRPCError(r.cli.Call("DatasetRequests.Remove", p, res))
	}
	ctx := context.TODO()

	log.Debugf("Remove dataset ref %q, revisions %v", p.Ref, p.Revision)

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
		log.Debugf("Remove, repo.CanonicalizeDatasetRef failed, error: %s", canonErr)
		if p.Force {
			didRemove, _ := base.RemoveEntireDataset(ctx, r.node.Repo, reporef.ConvertToDsref(ref), []DatasetLogItem{})
			if didRemove != "" {
				log.Debugf("Remove cleaned up data found in %s", didRemove)
				res.Message = didRemove
				return nil
			}
		}
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
					log.Debugf("Remove, IsWorkingDirectoryDirty")
					return ErrCantRemoveDirectoryDirty
				}
				if strings.Contains(wdErr.Error(), "not a linked directory") {
					// If the working directory has been removed (or renamed), could not get the
					// status. However, don't let this stop the remove operation, since the files
					// are already gone, and therefore won't be removed.
					log.Debugf("Remove, couldn't get status for %s, maybe removed or renamed", ref.FSIPath)
					wdErr = nil
				} else {
					log.Debugf("Remove, IsWorkingDirectoryClean error: %s", err)
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
	if err == nil && p.Revision.Gen >= len(history) {
		// If the number of revisions to delete is greater than or equal to the amount in history,
		// treat this operation as deleting everything.
		p.Revision.Gen = dsref.AllGenerations
	} else if err == repo.ErrNoHistory {
		// If the dataset has no history, treat this operation as deleting everything.
		p.Revision.Gen = dsref.AllGenerations
	} else if err != nil {
		log.Debugf("Remove, base.DatasetLog failed, error: %s", err)
		// Set history to a list of 0 elements. In the rest of this function, certain operations
		// check the history to figure out what to delete, they will always see a blank history,
		// which is a safer option for a destructive option such as remove.
		history = []DatasetLogItem{}
	}

	if p.Revision.Gen == dsref.AllGenerations {
		// removing all revisions of a dataset must unlink it
		if ref.FSIPath != "" {
			dr := reporef.ConvertToDsref(ref)
			if err := r.inst.fsi.Unlink(ref.FSIPath, dr); err == nil {
				res.Unlinked = true
			} else {
				log.Errorf("during Remove, dataset did not unlink: %s", err)
			}
		}

		didRemove, _ := base.RemoveEntireDataset(ctx, r.inst.Repo(), reporef.ConvertToDsref(ref), history)
		res.NumDeleted = dsref.AllGenerations
		res.Message = didRemove

		if ref.FSIPath != "" && !p.KeepFiles {
			// Remove all files
			fsi.DeleteComponentFiles(ref.FSIPath)
			var err error
			if p.Force {
				err = r.inst.fsi.RemoveAll(ref.FSIPath)
			} else {
				err = r.inst.fsi.Remove(ref.FSIPath)
			}
			if err != nil {
				if strings.Contains(err.Error(), "no such file or directory") {
					// If the working directory has already been removed (or renamed), it is
					// not an error that this remove operation fails, since we were trying to
					// remove them anyway.
					log.Debugf("Remove, couldn't remove %s, maybe already removed or renamed", ref.FSIPath)
					err = nil
				} else {
					log.Debugf("Remove, os.Remove failed, error: %s", err)
					return err
				}
			}
		}
	} else if len(history) > 0 {
		// Delete the specific number of revisions.
		hist := history[p.Revision.Gen]
		next := hist.SimpleRef()

		info, err := base.ModifyDatasetRef(ctx, r.node.Repo, reporef.ConvertToDsref(ref), next)
		if err != nil {
			log.Debugf("Remove, base.ModifyDatasetRef failed, error: %s", err)
			return err
		}
		newHead, err := base.RemoveNVersionsFromStore(ctx, r.inst.Repo(), reporef.ConvertToDsref(ref), p.Revision.Gen)
		if err != nil {
			log.Debugf("Remove, base.RemoveNVersionsFromStore failed, error: %s", err)
			return err
		}
		res.NumDeleted = p.Revision.Gen

		if info.FSIPath != "" && !p.KeepFiles {
			// Load dataset version that is at head after newer versions are removed
			ds, err := dsfs.LoadDataset(ctx, r.inst.Repo().Store(), newHead.Path)
			if err != nil {
				log.Debugf("Remove, dsfs.LoadDataset failed, error: %s", err)
				return err
			}
			ds.Name = newHead.Name
			ds.Peername = newHead.Username
			if err = base.OpenDataset(ctx, r.inst.Repo().Filesystem(), ds); err != nil {
				log.Debugf("Remove, base.OpenDataset failed, error: %s", err)
				return err
			}

			// TODO(dlong): Add a method to FSI called ProjectOntoDirectory, use it here
			// and also for Restore() in lib/fsi.go and also maybe WriteComponents in fsi/mapping.go

			// Delete the old files
			err = fsi.DeleteComponentFiles(info.FSIPath)
			if err != nil {
				log.Debug("Remove, fsi.DeleteComponentFiles failed, error: %s", err)
			}

			// Update the files in the working directory
			fsi.WriteComponents(ds, info.FSIPath, r.inst.node.Repo.Filesystem())
		}
	}
	log.Debugf("Remove finished")
	return nil
}

// AddParams encapsulates parameters to the add command
type AddParams struct {
	Ref        string
	LinkDir    string
	RemoteAddr string // remote to attempt to pull from
	LogsOnly   bool   // only fetch logbook data
}

// Add adds an existing dataset to a peer's repository
func (r *DatasetRequests) Add(p *AddParams, res *reporef.DatasetRef) (err error) {
	if err = qfs.AbsPath(&p.LinkDir); err != nil {
		return
	}

	if r.cli != nil {
		return checkRPCError(r.cli.Call("DatasetRequests.Add", p, res))
	}
	ctx := context.TODO()

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}

	if p.RemoteAddr == "" && r.inst != nil && r.inst.cfg.Registry != nil {
		p.RemoteAddr = r.inst.cfg.Registry.Location
	}

	mergeLogsError := r.inst.RemoteClient().CloneLogs(ctx, reporef.ConvertToDsref(ref), p.RemoteAddr)
	if p.LogsOnly {
		return mergeLogsError
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
}

// Validate gives a dataset of errors and issues for a given dataset
func (r *DatasetRequests) Validate(p *ValidateDatasetParams, valerrs *[]jsonschema.ValError) (err error) {
	if r.cli != nil {
		return checkRPCError(r.cli.Call("DatasetRequests.Validate", p, valerrs))
	}
	ctx := context.TODO()

	// TODO: restore validating data from a URL
	// if p.URL != "" && ref.IsEmpty() && o.Schema == nil {
	//   return (errors.New(ErrBadArgs, "if you are validating data from a url, please include a dataset name or supply the --schema flag with a file path that Qri can validate against"))
	// }

	// Schema can come from either schema.json or structure.json, or the dataset itself.
	// schemaFlagType determines which of these three contains the schema.
	schemaFlagType := ""
	schemaFilename := ""
	if p.SchemaFilename != "" && p.StructureFilename != "" {
		return errors.New(ErrBadArgs, "cannot provide both --schema and --structure flags")
	} else if p.SchemaFilename != "" {
		schemaFlagType = "schema"
		schemaFilename = p.SchemaFilename
	} else if p.StructureFilename != "" {
		schemaFlagType = "structure"
		schemaFilename = p.StructureFilename
	}

	if p.Ref == "" && (p.BodyFilename == "" || schemaFlagType == "") {
		return errors.New(ErrBadArgs, "please provide a dataset name, or a supply the --body and --schema or --structure flags")
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
		if ref.FSIPath != "" {
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

	*valerrs, err = base.Validate(ctx, r.node.Repo, body, st)
	return
}

// Manifest generates a manifest for a dataset path
func (r *DatasetRequests) Manifest(refstr *string, m *dag.Manifest) (err error) {
	if r.cli != nil {
		return checkRPCError(r.cli.Call("DatasetRequests.Manifest", refstr, m))
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
		return checkRPCError(r.cli.Call("DatasetRequests.Manifest", a, b))
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
		return checkRPCError(r.cli.Call("DatasetRequests.DAGInfo", s, i))
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
		return checkRPCError(r.cli.Call("DatasetRequests.Stats", p, res))
	}
	ctx := context.TODO()
	if p.Dataset == nil {
		ref := &reporef.DatasetRef{}
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
