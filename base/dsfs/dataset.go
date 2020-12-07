package dsfs

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/qfs"
)

// number of entries to per batch when processing body data in WriteDataset
const batchSize = 5000

var (
	// BodySizeSmallEnoughToDiff sets how small a body must be to generate a message from it
	BodySizeSmallEnoughToDiff = 20000000 // 20M or less is small
	// OpenFileTimeoutDuration determines the maximium amount of time to wait for
	// a Filestore to open a file. Some filestores (like IPFS) fallback to a
	// network request when it can't find a file locally. Setting a short timeout
	// prevents waiting for a slow network response, at the expense of leaving
	// files unresolved.
	// TODO (b5) - allow -1 duration as a sentinel value for no timeout
	OpenFileTimeoutDuration = time.Millisecond * 700
)

// If a user has a dataset larger than the above limit, then instead of diffing we compare the
// checksum against the previous version. We should make this algorithm agree with how `status`
// works.
// See issue: https://github.com/qri-io/qri/issues/1150

// LoadDataset reads a dataset from a cafs and dereferences structure, transform, and commitMsg if they exist,
// returning a fully-hydrated dataset
func LoadDataset(ctx context.Context, store qfs.Filesystem, path string) (*dataset.Dataset, error) {
	log.Debugf("LoadDataset path=%q", path)
	// set a timeout to handle long-lived requests when connected to IPFS.
	// if we don't have the dataset locally, IPFS will reach out onto the d.web to
	// attempt to resolve previous hashes. capping the duration yeilds quicker results.
	// TODO (b5) - The proper way to solve this is to feed a local-only IPFS store
	// to this entire function, or have a mechanism for specifying that a fetch
	// must be local
	ctx, cancel := context.WithTimeout(ctx, OpenFileTimeoutDuration)
	defer cancel()

	ds, err := LoadDatasetRefs(ctx, store, path)
	if err != nil {
		log.Debugf("loading dataset: %s", err)
		return nil, fmt.Errorf("loading dataset: %w", err)
	}
	if err := DerefDataset(ctx, store, ds); err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	return ds, nil
}

// LoadDatasetRefs reads a dataset from a content addressed filesystem without dereferencing
// it's components
func LoadDatasetRefs(ctx context.Context, fs qfs.Filesystem, path string) (*dataset.Dataset, error) {
	log.Debugf("LoadDatasetRefs path=%q", path)
	ds := dataset.NewDatasetRef(path)

	pathWithBasename := PackageFilepath(fs, path, PackageFileDataset)
	log.Debugf("getting %s", pathWithBasename)
	data, err := fileBytes(fs.Get(ctx, pathWithBasename))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("reading %s file: %w", PackageFileDataset.String(), err)
	}

	ds, err = dataset.UnmarshalDataset(data)
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("unmarshaling %s file: %w", PackageFileDataset.String(), err)
	}

	// assign path to retain internal reference to the path this dataset was read from
	ds.Path = path

	return ds, nil
}

// DerefDataset attempts to fully dereference a dataset
func DerefDataset(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	log.Debugf("DerefDataset path=%q", ds.Path)
	if err := DerefMeta(ctx, store, ds); err != nil {
		return err
	}
	if err := DerefStructure(ctx, store, ds); err != nil {
		return err
	}
	if err := DerefTransform(ctx, store, ds); err != nil {
		return err
	}
	if err := DerefViz(ctx, store, ds); err != nil {
		return err
	}
	if err := DerefReadme(ctx, store, ds); err != nil {
		return err
	}
	if err := DerefStats(ctx, store, ds); err != nil {
		return err
	}
	return DerefCommit(ctx, store, ds)
}

// SaveSwitches represents options for saving a dataset
type SaveSwitches struct {
	// Use a custom timestamp, defaults to time.Now if unset
	Time time.Time
	// Replace is whether the save is a full replacement or a set of patches to previous
	Replace bool
	// Pin is whether the dataset should be pinned
	Pin bool
	// ConvertFormatToPrev is whether the body should be converted to match the previous format
	ConvertFormatToPrev bool
	// ForceIfNoChanges is whether the save should be forced even if no changes are detected
	ForceIfNoChanges bool
	// ShouldRender is deprecated, controls whether viz should be rendered
	ShouldRender bool
	// NewName is whether a new dataset should be created, guaranteeing there's no previous version
	NewName bool
	// FileHint is a hint for what file is used for creating this dataset
	FileHint string
	// Drop is a string of components to remove before saving
	Drop string
}

// CreateDataset places a dataset into the store.
// Store is where we're going to store the data
// Dataset to be saved
// Prev is the previous version or nil if there isn't one
// Pk is the private key for cryptographically signing
// Sw is switches that control how the save happens
// Returns the immutable path if no error
func CreateDataset(
	ctx context.Context,
	source qfs.Filesystem,
	destination qfs.Filesystem,
	ds *dataset.Dataset,
	prev *dataset.Dataset,
	pk crypto.PrivKey,
	sw SaveSwitches,
) (string, error) {
	if pk == nil {
		return "", fmt.Errorf("private key is required to create a dataset")
	}

	if err := DerefDataset(ctx, source, ds); err != nil {
		log.Debugf("dereferencing dataset components: %s", err)
		return "", err
	}
	if err := validate.Dataset(ds); err != nil {
		log.Debug(err.Error())
		return "", err
	}
	log.Debugf("CreateDataset ds.Peername=%q ds.Name=%q writeDestType=%s", ds.Peername, ds.Name, destination.Type())

	if prev != nil && !prev.IsEmpty() {
		log.Debugw("dereferencing previous dataset", "prevPath", prev.Path)
		if err := DerefDataset(ctx, source, prev); err != nil {
			log.Debug(err.Error())
			return "", err
		}
		if err := validate.Dataset(prev); err != nil {
			log.Debug(err.Error())
			return "", err
		}
	}

	var (
		bf     = ds.BodyFile()
		bfPrev qfs.File
	)

	if prev != nil {
		bfPrev = prev.BodyFile()
	}
	if bf == nil && bfPrev == nil {
		return "", fmt.Errorf("bodyfile or previous bodyfile needed")
	} else if bf == nil {
		// TODO(dustmop): If no bf provided, we're assuming that the body is the same as it
		// was in the previous commit. In this case, we shouldn't be recalculating the
		// structure (err count, depth, checksum, length) we should just copy it from the
		// previous version.
		bf = bfPrev
	}

	// lock for editing dataset pointer
	var dsLk = &sync.Mutex{}

	bodyFile, err := newComputeFieldsFile(ctx, dsLk, source, pk, ds, prev, sw)
	if err != nil {
		return "", err
	}
	ds.SetBodyFile(bodyFile)

	path, err := WriteDataset(ctx, dsLk, destination, ds, pk, sw)
	if err != nil {
		log.Debug(err.Error())
		return "", err
	}

	// TODO (b5) - many codepaths that call this function use the `ds` arg after saving
	// we need to dereference here so fields are set, but this is overkill if
	// the caller doesn't use the ds arg afterward
	// might make sense to have a wrapper function that writes and loads on success
	if err := DerefDataset(ctx, destination, ds); err != nil {
		return path, err
	}
	return path, nil
}

// WriteDataset persists a datasets to a destination filesystem
func WriteDataset(
	ctx context.Context,
	dsLk *sync.Mutex,
	destination qfs.Filesystem,
	ds *dataset.Dataset,
	pk crypto.PrivKey,
	sw SaveSwitches,
) (string, error) {

	wfs := &writeFiles{
		// note: body ds.BodyFile() may return nil
		body: ds.BodyFile(),
	}

	// the call order of these functions is important, funcs later in the slice
	// may rely on writeFiles fields set by eariler functions
	addFuncs := []addWriteFileFunc{
		addMetaFile,
		addTransformFile,
		structureFileAddFunc(destination),
		addStatsFile,
		addReadmeFile,
		vizFilesAddFunc(destination, sw),
		commitFileAddFunc(pk),
		addDatasetFile,
	}

	for _, addFunc := range addFuncs {
		if err := addFunc(ds, wfs); err != nil {
			return "", err
		}
	}

	return qfs.WriteWithHooks(ctx, destination, wfs.root())
}

// writeFiles is a data structure for converting a dataset document into a set
// of qfs.File's. fields in writeFiles may be nil or a qfs.File
// many components pass back qfs.HookFiles that modify the contents of the
// dataset as it's being persisted
type writeFiles struct {
	meta qfs.File // no deps
	body qfs.File // no deps

	readmeScript    qfs.File // no deps
	vizScript       qfs.File // no deps
	transformScript qfs.File // no deps

	transform   qfs.File // requires transformScript if it exists
	structure   qfs.File // requires body if it exists
	stats       qfs.File // requires body, structure if they exist
	vizRendered qfs.File // requires body, meta, transform, structure, stats, readme if they exist

	commit  qfs.File // requires meta, transform, body, structure, stats, readme, vizScript, vizRendered if they exist
	dataset qfs.File // requires all other components
}

// files returns all non-nil files as a slice
func (wfs *writeFiles) files() []qfs.File {
	candidates := []qfs.File{
		wfs.meta,
		wfs.body,
		wfs.readmeScript,
		wfs.vizScript,
		wfs.transformScript,
		wfs.transform,
		wfs.structure,
		wfs.stats,
		wfs.vizRendered,
		wfs.commit,
		wfs.dataset,
	}

	files := make([]qfs.File, 0, len(candidates))
	for _, c := range candidates {
		if c != nil {
			files = append(files, c)
		}
	}
	return files
}

// root collects up all non-nil files in writeFiles wraps them in a root
// directory
func (wfs *writeFiles) root() qfs.File {
	return qfs.NewMemdir("/", wfs.files()...)
}

// addWriteFileFunc is a function that modifies one or more fields in a
// writeFiles struct by creating files from a given dataset
type addWriteFileFunc func(ds *dataset.Dataset, wfs *writeFiles) error

func addDatasetFile(ds *dataset.Dataset, wfs *writeFiles) error {
	pkgFiles := filePaths(wfs.files())
	if len(pkgFiles) == 0 {
		return fmt.Errorf("cannot save empty dataset")
	}
	log.Debugf("constructing dataset with pkgFiles=%v", pkgFiles)

	hook := func(ctx context.Context, f qfs.File, added map[string]string) (io.Reader, error) {
		ds.DropTransientValues()
		updateScriptPaths(ds, added)
		replaceComponentsWithRefs(ds, added, wfs.body.FullPath())

		if path, ok := added[PackageFileCommit.Filename()]; ok {
			ds.Commit = dataset.NewCommitRef(path)
		}

		return JSONFile(PackageFileDataset.Filename(), ds)
	}

	wfs.dataset = qfs.NewWriteHookFile(emptyFile(PackageFileDataset.Filename()), hook, filePaths(wfs.files())...)
	return nil
}

func filePaths(fs []qfs.File) (files []string) {
	for _, f := range fs {
		files = append(files, f.FullPath())
	}
	return files
}

func emptyFile(fullPath string) qfs.File {
	return qfs.NewMemfileBytes(fullPath, []byte{})
}

func updateScriptPaths(ds *dataset.Dataset, added map[string]string) {
	for filepath, addr := range added {
		switch filepath {
		case PackageFileVizScript.Filename():
			ds.Viz.ScriptPath = addr
		case PackageFileRenderedViz.Filename():
			ds.Viz.RenderedPath = addr
		case PackageFileReadmeScript.Filename():
			ds.Readme.ScriptPath = addr
		}
	}
}

func replaceComponentsWithRefs(ds *dataset.Dataset, added map[string]string, bodyPathName string) {
	for filepath, addr := range added {
		switch filepath {
		case PackageFileStructure.Filename():
			ds.Structure = dataset.NewStructureRef(addr)
		case PackageFileViz.Filename():
			ds.Viz = dataset.NewVizRef(addr)
		case PackageFileMeta.Filename():
			ds.Meta = dataset.NewMetaRef(addr)
		case PackageFileStats.Filename():
			ds.Stats = dataset.NewStatsRef(addr)
		case bodyPathName:
			ds.BodyPath = addr
		}
	}
}
