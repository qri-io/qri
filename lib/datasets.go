package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/qri-io/dag"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/archive"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/base/fill"
	"github.com/qri-io/qri/dscache/build"
	"github.com/qri-io/qri/dsref"
	qrierr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/fsi/linkfile"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

// DatasetMethods encapsulates business logic for working with Datasets on Qri
type DatasetMethods struct {
	inst *Instance
}

// CoreRequestsName implements the Requets interface
func (DatasetMethods) CoreRequestsName() string { return "datasets" }

// NewDatasetMethods creates a DatasetMethods pointer from a qri instance
func NewDatasetMethods(inst *Instance) *DatasetMethods {
	return &DatasetMethods{
		inst: inst,
	}
}

// List gets the reflist for either the local repo or a peer
func (m *DatasetMethods) List(p *ListParams, res *[]dsref.VersionInfo) error {
	if m.inst.rpc != nil {
		p.RPC = true
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.List", p, res))
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
	if err := repo.CanonicalizeProfile(m.inst.repo, ref); err != nil {
		return fmt.Errorf("error canonicalizing peer: %s", err.Error())
	}

	pro, err := m.inst.repo.Profile()
	if err != nil {
		return err
	}

	var refs []reporef.DatasetRef
	if p.UseDscache {
		c := m.inst.dscache
		if c.IsEmpty() {
			log.Infof("building dscache from repo's logbook, profile, and dsref")
			built, err := build.DscacheFromRepo(ctx, m.inst.repo)
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
		refs, err = base.ListDatasets(ctx, m.inst.repo, p.Term, p.Limit, p.Offset, p.RPC, p.Published, p.ShowNumVersions)
	} else {

		refs, err = m.inst.remoteClient.ListDatasets(ctx, ref, p.Term, p.Offset, p.Limit)
	}
	if err != nil {
		return err
	}

	if p.EnsureFSIExists {
		// For each reference with a linked fsi working directory, check that the folder exists
		// and has a .qri-ref file. If it's missing, remove the link from the centralized repo.
		// Doing this every list operation is a bit inefficient, so the behavior is opt-in.
		for _, ref := range refs {
			if ref.FSIPath != "" && !linkfile.ExistsInDir(ref.FSIPath) {
				ref.FSIPath = ""
				if ref.Path == "" {
					if err = m.inst.repo.DeleteRef(ref); err != nil {
						log.Debugf("cannot delete ref for %q, err: %s", ref, err)
					}
					continue
				}
				if err = m.inst.repo.PutRef(ref); err != nil {
					log.Debugf("cannot put ref for %q, err: %s", ref, err)
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
func (m *DatasetMethods) ListRawRefs(p *ListParams, text *string) error {
	var err error
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.ListRawRefs", p, text))
	}
	ctx := context.TODO()
	if p.UseDscache {
		c := m.inst.dscache
		if c == nil || c.IsEmpty() {
			return fmt.Errorf("repo: dscache not found")
		}
		*text = c.VerboseString(true)
		return nil
	}

	*text, err = base.RawDatasetRefs(ctx, m.inst.repo)
	return err
}

// GetParams defines parameters for looking up the head or body of a dataset
type GetParams struct {
	// Refstr to get, representing a dataset ref to be parsed
	Refstr   string
	Selector string

	// read from a filesystem link instead of stored version
	Format       string
	FormatConfig dataset.FormatConfig

	Limit, Offset int
	All           bool

	// outfile is a filename to save the dataset to
	Outfile string
	// whether to generate a filename from the dataset name instead
	GenFilename bool
	Remote      string
}

// GetResult combines data with it's hashed path
type GetResult struct {
	Ref       *dsref.Ref       `json:"ref"`
	Dataset   *dataset.Dataset `json:"data"`
	Bytes     []byte           `json:"bytes"`
	Message   string           `json:"message"`
	FSIPath   string           `json:"fsipath"`
	Published bool             `json:"published"`
}

// Get retrieves datasets and components for a given reference. p.Refstr is parsed to create
// a reference, which is used to load the dataset. It will be loaded from the local repo
// or from the filesystem if it has a linked working direoctry.
// Using p.Selector will control what components are returned in res.Bytes. The default,
// a blank selector, will also fill the entire dataset at res.Data. If the selector is "body"
// then res.Bytes is loaded with the body. If the selector is "stats", then res.Bytes is loaded
// with the generated stats.
func (m *DatasetMethods) Get(p *GetParams, res *GetResult) error {
	if err := qfs.AbsPath(&p.Outfile); err != nil {
		return err
	}

	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.Get", p, res))
	}
	ctx := context.TODO()

	var ds *dataset.Dataset
	ref, source, err := m.inst.ParseAndResolveRefWithWorkingDir(ctx, p.Refstr, p.Remote)
	if err != nil {
		return err
	}
	ds, err = m.inst.LoadDataset(ctx, ref, source)
	if err != nil {
		return err
	}

	res.Ref = &ref
	res.Dataset = ds
	// TODO (b5) - FSIPath is determined differently now: by checking .Path for
	// an /fsi prefix
	// TODO (b5) - Published field is longer set as part of Reference Resolution
	// getting publication status should be delegated to a new function

	if err = base.OpenDataset(ctx, m.inst.repo.Filesystem(), ds); err != nil {
		log.Debugf("Get dataset, base.OpenDataset failed, error: %s", err)
		return err
	}

	if p.Format == "zip" {
		// Only if GenFilename is true, and no output filename is set, generate one from the
		// dataset name
		if p.Outfile == "" && p.GenFilename {
			p.Outfile = fmt.Sprintf("%s.zip", ds.Name)
		}
		var outBuf bytes.Buffer
		var zipFile io.Writer
		if p.Outfile == "" {
			// In this case, write to a buffer, which will be assigned to res.Bytes later on
			zipFile = &outBuf
		} else {
			zipFile, err = os.Create(p.Outfile)
			if err != nil {
				return err
			}
		}
		currRef := dsref.Ref{Username: ds.Peername, Name: ds.Name}
		// TODO(dustmop): This function is inefficient and a poor use of logbook, but it's
		// necessary until dscache is in use.
		initID, err := m.inst.repo.Logbook().RefToInitID(currRef)
		if err != nil {
			return err
		}
		err = archive.WriteZip(ctx, m.inst.repo.Store(), ds, "json", initID, currRef, zipFile)
		if err != nil {
			return err
		}
		// Handle output. If outfile is empty, return the raw bytes. Otherwise provide a helpful
		// message for the user
		if p.Outfile == "" {
			res.Bytes = outBuf.Bytes()
		} else {
			res.Message = fmt.Sprintf("Wrote archive %s", p.Outfile)
		}
		return nil
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

		// TODO (b5) - replace this prefix check with a call to qfs.PathKind when it
		// supports the fsi prefix
		if strings.HasPrefix(ref.Path, "/fsi") {
			// TODO(dustmop): Need to handle the special case where an FSI directory has a body
			// but no structure, which should infer a schema in order to read the body. Once that
			// works we can remove the fsi.GetBody call and just use base.ReadBody.
			res.Bytes, err = fsi.GetBody(strings.TrimPrefix(ref.Path, "/fsi"), df, p.FormatConfig, p.Offset, p.Limit, p.All)
			if err != nil {
				log.Debugf("Get dataset, fsi.GetBody %q failed, error: %s", res.FSIPath, err)
				return err
			}
			return m.maybeWriteOutfile(p, res)
		}
		res.Bytes, err = base.ReadBody(ds, df, p.FormatConfig, p.Limit, p.Offset, p.All)
		if err != nil {
			log.Debugf("Get dataset, base.ReadBody %q failed, error: %s", ds, err)
			return err
		}
		return m.maybeWriteOutfile(p, res)
	} else if scriptFile, ok := scriptFileSelection(ds, p.Selector); ok {
		// Fields that have qfs.File types should be read and returned
		res.Bytes, err = ioutil.ReadAll(scriptFile)
		if err != nil {
			return err
		}
		return m.maybeWriteOutfile(p, res)
	} else if p.Selector == "stats" {
		statsParams := &StatsParams{
			Dataset: res.Dataset,
		}
		statsRes := &StatsResponse{}
		if err = m.Stats(statsParams, statsRes); err != nil {
			return err
		}
		res.Bytes = statsRes.StatsBytes
		return m.maybeWriteOutfile(p, res)
	}
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
	return m.maybeWriteOutfile(p, res)
}

func (m *DatasetMethods) maybeWriteOutfile(p *GetParams, res *GetResult) error {
	if p.Outfile != "" {
		err := ioutil.WriteFile(p.Outfile, res.Bytes, 0644)
		if err != nil {
			return err
		}
		res.Bytes = []byte{}
	}
	return nil
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
	// comma separated list of component names to delete before saving
	Drop string
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
func (m *DatasetMethods) Save(p *SaveParams, res *reporef.DatasetRef) error {
	if m.inst.rpc != nil {
		p.ScriptOutput = nil
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.Save", p, res))
	}
	ctx := context.TODO()

	if p.Private {
		return fmt.Errorf("option to make dataset private not yet implemented, refer to https://github.com/qri-io/qri/issues/291 for updates")
	}

	ref, err := dsref.ParseHumanFriendly(p.Ref)
	if errors.Is(err, dsref.ErrBadCaseName) {
		// If dataset name is using bad-case characters, and is not yet in use, fail with error.
		if !m.nameIsInUse(ctx, ref) {
			return err
		}
		// If dataset name already exists, just log a warning and then continue.
		log.Error(dsref.ErrBadCaseShouldRename)
	} else if errors.Is(err, dsref.ErrEmptyRef) {
		// Okay if reference is empty. Later code will try to infer the name from other parameters.
	} else if errors.Is(err, dsref.ErrNotHumanFriendly) {
		return err
	} else if err != nil {
		// If some other parse error happened, describe a valid dataset name.
		return dsref.ErrDescribeValidName
	}

	// Validate that username is our own, it's not valid to try to save a dataset with someone
	// else's username. Without this check, base will replace the username with our own regardless,
	// it's better to have an error to display, rather than silently ignore it.
	pro, err := m.inst.repo.Profile()
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

	// TODO(dustmop): In the future, resolve the dataset ref early, get the initID, and use that
	// everywhere. If dataset has no name, don't bother to canonicalize, instead, call Infer
	// here. If either ref failed to resolve, or Infer was called, generate a new initID using
	// logbook immediately. Regardless, stop using the dsref after this point.

	// Check if the dataset has an FSIPath, which requires a different save codepath.
	err = repo.CanonicalizeDatasetRef(m.inst.repo, &datasetRef)
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
		recall, err := base.Recall(ctx, m.inst.repo, p.Recall, datasetRef)
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
	if !p.Force && p.Drop == "" &&
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

	if err = base.OpenDataset(ctx, m.inst.repo.Filesystem(), ds); err != nil {
		log.Debugf("open ds error: %s", err.Error())
		return err
	}

	// If a transform is being provided, execute its script
	// TODO(dustmop): This will become a call to `apply` in the future, and will require the
	// `--apply` flag to be true.
	if ds.Transform != nil {
		str := m.inst.node.LocalStreams
		scriptOut := p.ScriptOutput
		secrets := p.Secrets
		r := m.inst.repo
		if p.DryRun {
			str.PrintErr("🏃🏽‍♀️ dry run\n")
			// dry-runs store to an in-memory repo
			r, err = repo.NewMemRepo(pro, cafs.NewMapstore(), r.Filesystem(), profile.NewMemStore())
			if err != nil {
				log.Debugf("creating new memRepo: %s", err)
				return nil
			}
		}

		// create a loader so transforms can call `load_dataset`
		// TODO(b5) - add a ResolverMode save parameter and call m.inst.resolverMode
		// on the passed in mode string instead of just using the default resolver
		// cmd can then define "remote" and "offline" flags, that set the ResolverMode
		// string and control how transform functions
		loader := dsref.NewParseResolveLoadFunc("", m.inst.defaultResolver(), m.inst)

		// apply the transform
		err := base.TransformApply(ctx, ds, r, loader, str, scriptOut, secrets)
		if err != nil {
			return err
		}
	}

	if p.DryRun {
		// Tests expect a that a call to `qri save --dry-run` will still construct a full
		// reference with an IPFS path and Name, etc. This isn't actually a valid reference,
		// since nothing is written to the repo, so relying on this is a bit hacky. But using
		// dry-run to save is going away once `apply` exists, so this is temporary anyway.
		*res = reporef.DatasetRef{
			ProfileID: pro.ID,
			Name:      ds.Name,
			Peername:  pro.Peername,
			Path:      ds.Path,
			Dataset:   ds,
		}
		return nil
	}

	// If the dscache doesn't exist yet, it will only be created if the appropriate flag enables it.
	if p.UseDscache && !p.DryRun {
		c := m.inst.dscache
		c.CreateNewEnabled = true
	}

	// TODO (b5) - this should be integrated into base.SaveDataset
	fsiPath := datasetRef.FSIPath

	if fsiPath != "" && p.Drop != "" {
		return qrierr.New(fmt.Errorf("cannot drop while FSI-linked"), "can't drop component from a working-directory, delete files instead.")
	}

	fileHint := p.BodyPath
	if len(p.FilePaths) > 0 {
		fileHint = p.FilePaths[0]
	}

	// Determine dataset name (inferring one for a blank name), and lookup the initID for that
	// name. Also get the path if the dataset has an existing version.
	trueRef, err := base.FinalizeNameAndStableIdentifers(ctx, m.inst.repo, pro.Peername, ds.Name, ds, p.NewName)
	if err != nil {
		return err
	}
	// TODO(dustmop): This is needed temporarily, as base.CreateDataset needs the name field in
	// order to update the refstore. dsfs will clear this field before the dataset is written to
	// IPFS. Once everything switches to logbook and dscache, this field assignment will no longer
	// be needed.
	ds.Name = trueRef.Name

	switches := base.SaveSwitches{
		FileHint:            fileHint,
		Replace:             p.Replace,
		Pin:                 true,
		ConvertFormatToPrev: p.ConvertFormatToPrev,
		ForceIfNoChanges:    p.Force,
		ShouldRender:        p.ShouldRender,
		NewName:             p.NewName,
		Drop:                p.Drop,
	}
	datasetRef, err = base.SaveDataset(ctx, m.inst.repo, trueRef.InitID, trueRef.Path, ds, switches)
	if err != nil {
		log.Debugf("create ds error: %s\n", err.Error())
		return err
	}

	// TODO (b5) - this should be integrated into base.SaveDataset
	if fsiPath != "" && !p.DryRun {
		datasetRef.FSIPath = fsiPath
		if err = m.inst.repo.PutRef(datasetRef); err != nil {
			return err
		}
	}

	if p.ReturnBody {
		if err = base.InlineJSONBody(datasetRef.Dataset); err != nil {
			return err
		}
	}

	if p.Publish {
		if p.DryRun {
			return fmt.Errorf("can't use publish & dry-run together")
		}
		var publishedRef reporef.DatasetRef
		err = m.SetPublishStatus(&SetPublishStatusParams{
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

	if fsiPath != "" && !p.DryRun {
		// Need to pass filesystem here so that we can read the README component and write it
		// properly back to disk.
		fsi.WriteComponents(res.Dataset, datasetRef.FSIPath, m.inst.repo.Filesystem())
	}
	return nil
}

// This is somewhat of a hack, we shouldn't need to lookup anything about the dataset reference
// before running Save. However, we need to check for now until we solve the problem of
// dataset names existing with bad-case characters.
// See this issue: https://github.com/qri-io/qri/issues/1132
func (m *DatasetMethods) nameIsInUse(ctx context.Context, ref dsref.Ref) bool {
	res := ref.Copy()
	_, err := m.inst.ResolveReference(ctx, &res, "local")
	if errors.Is(err, dsref.ErrRefNotFound) {
		return false
	} else if err != nil {
		// TODO(b5): Unsure if this is correct. If `Get` hits some other error, we aren't
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
func (m *DatasetMethods) SetPublishStatus(p *SetPublishStatusParams, publishedRef *reporef.DatasetRef) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.SetPublishStatus", p, publishedRef))
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(m.inst.repo, &ref); err != nil {
		return err
	}

	ref.Published = p.PublishStatus
	if err = base.SetPublishStatus(m.inst.repo, &ref, ref.Published); err != nil {
		return err
	}

	*publishedRef = ref
	return nil
}

// RenameParams defines parameters for Dataset renaming
type RenameParams struct {
	Current, Next string
}

// Rename changes a user's given name for a dataset
func (m *DatasetMethods) Rename(p *RenameParams, res *dsref.VersionInfo) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.Rename", p, res))
	}
	ctx := context.TODO()

	if p.Current == "" {
		return fmt.Errorf("current name is required to rename a dataset")
	}

	ref, err := dsref.ParseHumanFriendly(p.Current)
	// Allow bad upper-case characters in the left-hand side name, because it's needed to let users
	// fix badly named datasets.
	if err != nil && err != dsref.ErrBadCaseName {
		return fmt.Errorf("original name: %s", err)
	}
	if _, err := m.inst.ResolveReference(ctx, &ref, "local"); err != nil {
		return err
	}

	next, err := dsref.ParseHumanFriendly(p.Next)
	if errors.Is(err, dsref.ErrNotHumanFriendly) {
		return fmt.Errorf("destination name: %s", err)
	} else if err != nil {
		return fmt.Errorf("destination name: %s", dsref.ErrDescribeValidName)
	}
	if ref.Username != next.Username && next.Username != "me" {
		return fmt.Errorf("cannot change username or profileID of a dataset")
	}

	// Update the reference stored in the repo
	vi, err := base.RenameDatasetRef(ctx, m.inst.repo, ref, next.Name)
	if err != nil {
		return err
	}

	// If the dataset is linked to a working directory, update the ref
	if vi.FSIPath != "" {
		if _, err = m.inst.fsi.ModifyLinkReference(vi.FSIPath, vi.SimpleRef()); err != nil {
			return err
		}
	}

	*res = *vi
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
func (m *DatasetMethods) Remove(p *RemoveParams, res *RemoveResponse) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.Remove", p, res))
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

	if canonErr := repo.CanonicalizeDatasetRef(m.inst.repo, &ref); canonErr != nil && canonErr != repo.ErrNoHistory {
		log.Debugf("Remove, repo.CanonicalizeDatasetRef failed, error: %s", canonErr)
		if p.Force {
			didRemove, _ := base.RemoveEntireDataset(ctx, m.inst.repo, reporef.ConvertToDsref(ref), []DatasetLogItem{})
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
			wdErr := m.inst.fsi.IsWorkingDirectoryClean(ctx, ref.FSIPath)
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
	history, err := base.DatasetLog(ctx, m.inst.repo, reporef.ConvertToDsref(ref), p.Revision.Gen+1, 0, false)
	if err == nil && p.Revision.Gen >= len(history) {
		// If the number of revisions to delete is greater than or equal to the amount in history,
		// treat this operation as deleting everything.
		p.Revision.Gen = dsref.AllGenerations
	} else if err == repo.ErrNoHistory || errors.Is(err, dsref.ErrPathRequired) {
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
			if err := m.inst.fsi.Unlink(ref.FSIPath, dr); err == nil {
				res.Unlinked = true
			} else {
				log.Errorf("during Remove, dataset did not unlink: %s", err)
			}
		}

		didRemove, _ := base.RemoveEntireDataset(ctx, m.inst.repo, reporef.ConvertToDsref(ref), history)
		res.NumDeleted = dsref.AllGenerations
		res.Message = didRemove

		if ref.FSIPath != "" && !p.KeepFiles {
			// Remove all files
			fsi.DeleteComponentFiles(ref.FSIPath)
			var err error
			if p.Force {
				err = m.inst.fsi.RemoveAll(ref.FSIPath)
			} else {
				err = m.inst.fsi.Remove(ref.FSIPath)
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
		info, err := base.RemoveNVersionsFromStore(ctx, m.inst.repo, reporef.ConvertToDsref(ref), p.Revision.Gen)
		if err != nil {
			log.Debugf("Remove, base.RemoveNVersionsFromStore failed, error: %s", err)
			return err
		}
		res.NumDeleted = p.Revision.Gen

		if info.FSIPath != "" && !p.KeepFiles {
			// Load dataset version that is at head after newer versions are removed
			ds, err := dsfs.LoadDataset(ctx, m.inst.repo.Store(), info.Path)
			if err != nil {
				log.Debugf("Remove, dsfs.LoadDataset failed, error: %s", err)
				return err
			}
			ds.Name = info.Name
			ds.Peername = info.Username
			if err = base.OpenDataset(ctx, m.inst.repo.Filesystem(), ds); err != nil {
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
			fsi.WriteComponents(ds, info.FSIPath, m.inst.repo.Filesystem())
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
func (m *DatasetMethods) Add(p *AddParams, res *reporef.DatasetRef) error {
	if err := qfs.AbsPath(&p.LinkDir); err != nil {
		return err
	}

	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.Add", p, res))
	}
	ctx := context.TODO()

	ref, err := dsref.Parse(p.Ref)
	if err != nil {
		return err
	}

	if p.RemoteAddr == "" && m.inst != nil && m.inst.cfg.Registry != nil {
		p.RemoteAddr = m.inst.cfg.Registry.Location
	}

	mergeLogsError := m.inst.remoteClient.CloneLogs(ctx, ref, p.RemoteAddr)
	// TODO(b5) - this line is swallowing errors that the cmd package integration
	// tests are hitting. We need to change the behaviour of add to *require* logs
	// successfully merge, which will require fixing lots of tests in cmd,
	// ideally by removing remote.MockRepoClient
	if p.LogsOnly {
		return mergeLogsError
	}

	rref := reporef.RefFromDsref(ref)
	if err = m.inst.remoteClient.AddDataset(ctx, &rref, p.RemoteAddr); err != nil {
		return err
	}

	*res = rref

	if p.LinkDir != "" {
		checkoutp := &CheckoutParams{
			Ref: ref.String(),
			Dir: p.LinkDir,
		}
		m := NewFSIMethods(m.inst)
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
func (m *DatasetMethods) Validate(p *ValidateDatasetParams, valerrs *[]jsonschema.KeyError) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.Validate", p, valerrs))
	}
	ctx := context.TODO()

	// TODO: restore validating data from a URL
	// if p.URL != "" && ref.IsEmpty() && o.Schema == nil {
	//   return (qrierr.New(ErrBadArgs, "if you are validating data from a url, please include a dataset name or supply the --schema flag with a file path that Qri can validate against"))
	// }

	// Schema can come from either schema.json or structure.json, or the dataset itself.
	// schemaFlagType determines which of these three contains the schema.
	schemaFlagType := ""
	schemaFilename := ""
	if p.SchemaFilename != "" && p.StructureFilename != "" {
		return qrierr.New(ErrBadArgs, "cannot provide both --schema and --structure flags")
	} else if p.SchemaFilename != "" {
		schemaFlagType = "schema"
		schemaFilename = p.SchemaFilename
	} else if p.StructureFilename != "" {
		schemaFlagType = "structure"
		schemaFilename = p.StructureFilename
	}

	if p.Ref == "" && (p.BodyFilename == "" || schemaFlagType == "") {
		return qrierr.New(ErrBadArgs, "please provide a dataset name, or a supply the --body and --schema or --structure flags")
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil && err != repo.ErrEmptyRef {
		return err
	}
	err = repo.CanonicalizeDatasetRef(m.inst.repo, &ref)
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
			if ds, err = dsfs.LoadDataset(ctx, m.inst.repo.Store(), ref.Path); err != nil {
				return fmt.Errorf("loading dataset: %s", err)
			}
		}
		if err = base.OpenDataset(ctx, m.inst.repo.Filesystem(), ds); err != nil {
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
			return fmt.Errorf("error opening %s file: %s", schemaFlagType, schemaFilename)
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

	*valerrs, err = base.Validate(ctx, m.inst.repo, body, st)
	return err
}

// Manifest generates a manifest for a dataset path
func (m *DatasetMethods) Manifest(refstr *string, mfst *dag.Manifest) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.Manifest", refstr, mfst))
	}
	ctx := context.TODO()

	ref, err := repo.ParseDatasetRef(*refstr)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(m.inst.repo, &ref); err != nil {
		return err
	}

	var mf *dag.Manifest
	mf, err = m.inst.node.NewManifest(ctx, ref.Path)
	if err != nil {
		return err
	}
	*mfst = *mf
	return nil
}

// ManifestMissing generates a manifest of blocks that are not present on this repo for a given manifest
func (m *DatasetMethods) ManifestMissing(a, b *dag.Manifest) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.Manifest", a, b))
	}
	ctx := context.TODO()

	var mf *dag.Manifest
	mf, err := m.inst.node.MissingManifest(ctx, a)
	if err != nil {
		return err
	}
	*b = *mf
	return nil
}

// DAGInfoParams defines parameters for the DAGInfo method
type DAGInfoParams struct {
	RefStr, Label string
}

// DAGInfo generates a dag.Info for a dataset path. If a label is given, DAGInfo will generate a sub-dag.Info at that label.
func (m *DatasetMethods) DAGInfo(s *DAGInfoParams, i *dag.Info) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.DAGInfo", s, i))
	}
	ctx := context.TODO()

	ref, err := repo.ParseDatasetRef(s.RefStr)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(m.inst.repo, &ref); err != nil {
		return err
	}

	var info *dag.Info
	info, err = m.inst.node.NewDAGInfo(ctx, ref.Path, s.Label)
	if err != nil {
		return err
	}
	*i = *info
	return err
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
func (m *DatasetMethods) Stats(p *StatsParams, res *StatsResponse) error {
	var err error
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("DatasetMethods.Stats", p, res))
	}
	ctx := context.TODO()

	if p.Dataset == nil {
		// TODO (b5) - stats is currently local-only, supply a source parameter
		ref, source, err := m.inst.ParseAndResolveRefWithWorkingDir(ctx, p.Ref, "local")
		if err != nil {
			return err
		}
		p.Dataset, err = m.inst.LoadDataset(ctx, ref, source)
		if err != nil {
			return fmt.Errorf("loading dataset: %w", err)
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
		if err = p.Dataset.OpenBodyFile(ctx, m.inst.repo.Filesystem()); err != nil {
			return err
		}
	}
	reader, err := m.inst.stats.JSON(ctx, p.Dataset)
	if err != nil {
		return err
	}
	res.StatsBytes, err = ioutil.ReadAll(reader)
	return err
}
