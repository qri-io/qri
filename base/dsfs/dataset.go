package dsfs

import (
	"context"
	"encoding/json"
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
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading dataset: %s", err.Error())
	}
	if err := DerefDataset(ctx, store, ds); err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	return ds, nil
}

// LoadDatasetRefs reads a dataset from a content addressed filesystem without dereferencing
// it's components
func LoadDatasetRefs(ctx context.Context, store qfs.Filesystem, path string) (*dataset.Dataset, error) {
	log.Debugf("LoadDatasetRefs path=%q", path)
	ds := dataset.NewDatasetRef(path)

	pathWithBasename := PackageFilepath(store, path, PackageFileDataset)
	log.Debugf("getting %s", pathWithBasename)
	data, err := fileBytes(store.Get(ctx, pathWithBasename))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error getting file bytes: %s", err.Error())
	}

	ds, err = dataset.UnmarshalDataset(data)
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error unmarshaling %s file: %s", PackageFileDataset.String(), err.Error())
	}

	// assign path to retain internal reference to the
	// path this dataset was read from
	ds.Assign(dataset.NewDatasetRef(path))

	return ds, nil
}

// DerefDataset attempts to fully dereference a dataset
func DerefDataset(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	log.Debugf("DerefDataset path=%q", ds.Path)
	if err := DerefDatasetMeta(ctx, store, ds); err != nil {
		return err
	}
	if err := DerefDatasetStructure(ctx, store, ds); err != nil {
		return err
	}
	if err := DerefDatasetTransform(ctx, store, ds); err != nil {
		return err
	}
	if err := DerefDatasetViz(ctx, store, ds); err != nil {
		return err
	}
	if err := DerefDatasetReadme(ctx, store, ds); err != nil {
		return err
	}
	return DerefDatasetCommit(ctx, store, ds)
}

// DerefDatasetStructure derferences a dataset's structure element if required
// should be a no-op if ds.Structure is nil or isn't a reference
func DerefDatasetStructure(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Structure != nil && ds.Structure.IsEmpty() && ds.Structure.Path != "" {
		st, err := loadStructure(ctx, store, ds.Structure.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error loading dataset structure: %s", err.Error())
		}
		// assign path to retain internal reference to path
		// st.Assign(dataset.NewStructureRef(ds.Structure.Path))
		ds.Structure = st
	}
	return nil
}

// DerefDatasetViz dereferences a dataset's Viz element if required
// should be a no-op if ds.Viz is nil or isn't a reference
func DerefDatasetViz(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Viz != nil && ds.Viz.IsEmpty() && ds.Viz.Path != "" {
		vz, err := loadViz(ctx, store, ds.Viz.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error loading dataset viz: %s", err.Error())
		}
		// assign path to retain internal reference to path
		// vz.Assign(dataset.NewVizRef(ds.Viz.Path))
		ds.Viz = vz
	}
	return nil
}

// DerefDatasetReadme dereferences a dataset's Readme element if required
// should be a no-op if ds.Readme is nil or isn't a reference
func DerefDatasetReadme(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Readme != nil && ds.Readme.IsEmpty() && ds.Readme.Path != "" {
		rm, err := loadReadme(ctx, store, ds.Readme.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error loading dataset readme: %s", err.Error())
		}
		// assign path to retain internal reference to path
		// rm.Assign(dataset.NewVizRef(ds.Readme.Path))
		ds.Readme = rm
	}
	return nil
}

// DerefDatasetTransform derferences a dataset's transform element if required
// should be a no-op if ds.Structure is nil or isn't a reference
func DerefDatasetTransform(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Transform != nil && ds.Transform.IsEmpty() && ds.Transform.Path != "" {
		t, err := loadTransform(ctx, store, ds.Transform.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error loading dataset transform: %s", err.Error())
		}
		// assign path to retain internal reference to path
		// t.Assign(dataset.NewTransformRef(ds.Transform.Path))
		ds.Transform = t
	}
	return nil
}

// DerefDatasetMeta derferences a dataset's transform element if required
// should be a no-op if ds.Structure is nil or isn't a reference
func DerefDatasetMeta(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Meta != nil && ds.Meta.IsEmpty() && ds.Meta.Path != "" {
		md, err := loadMeta(ctx, store, ds.Meta.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error loading dataset metadata: %s", err.Error())
		}
		// assign path to retain internal reference to path
		// md.Assign(dataset.NewMetaRef(ds.Meta.Path))
		ds.Meta = md
	}
	return nil
}

// DerefDatasetCommit derferences a dataset's Commit element if required
// should be a no-op if ds.Structure is nil or isn't a reference
func DerefDatasetCommit(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Commit != nil && ds.Commit.IsEmpty() && ds.Commit.Path != "" {
		cm, err := loadCommit(ctx, store, ds.Commit.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error loading dataset commit: %s", err.Error())
		}
		// assign path to retain internal reference to path
		cm.Assign(dataset.NewCommitRef(ds.Commit.Path))
		ds.Commit = cm
	}
	return nil
}

// SaveSwitches represents options for saving a dataset
type SaveSwitches struct {
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
	ds, prev *dataset.Dataset,
	pk crypto.PrivKey,
	sw SaveSwitches,
) (string, error) {
	if pk == nil {
		return "", fmt.Errorf("private key is required to create a dataset")
	}
	if err := DerefDataset(ctx, source, ds); err != nil {
		log.Debug(err.Error())
		return "", err
	}
	if err := validate.Dataset(ds); err != nil {
		log.Debug(err.Error())
		return "", err
	}
	log.Debugf("CreateDataset ds.Peername=%q ds.Name=%q", ds.Peername, ds.Name)

	if prev != nil && !prev.IsEmpty() {
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
	return path, nil
}

func WriteDataset(
	ctx context.Context,
	dsLk *sync.Mutex,
	destination qfs.Filesystem,
	ds *dataset.Dataset,
	pk crypto.PrivKey,
	sw SaveSwitches,
) (string, error) {

	root, err := buildFileGraph(destination, ds, sw)
	if err != nil {
		return "", err
	}

	return qfs.WriteWithHooks(ctx, destination, root)
}

func buildFileGraph(fs qfs.Filesystem, ds *dataset.Dataset, sw SaveSwitches) (root qfs.File, err error) {
	var (
		components []string
		files      []qfs.File
		bdf        = ds.BodyFile()
		// TODO (b5) - pathing needs more work
		bodyFullPath string
	)

	if bdf != nil {
		bodyFullPath = fmt.Sprintf("/%s", bdf.FullPath())
		files = append(files, bdf)
		components = append(components, bodyFullPath)
	}

	if ds.Structure != nil {
		ds.Structure.DropTransientValues()

		stf, err := JSONFile(PackageFileStructure.Filename(), ds.Structure)
		if err != nil {
			return nil, err
		}

		if bdf != nil {
			hook := func(ctx context.Context, f qfs.File, added map[string]string) (io.Reader, error) {
				return JSONFile(f.FullPath(), ds.Structure)
			}
			stf = qfs.NewWriteHookFile(stf, hook, bodyFullPath)
		}

		files = append(files, stf)
		components = append(components, PackageFileStructure.Filename())
	}

	if ds.Meta != nil {
		ds.Meta.DropTransientValues()
		mdf, err := JSONFile(PackageFileMeta.Filename(), ds.Meta)
		if err != nil {
			return nil, fmt.Errorf("encoding meta component to json: %w", err)
		}
		files = append(files, mdf)
		components = append(components, PackageFileMeta.Filename())
	}

	if ds.Transform != nil {
		ds.Transform.DropTransientValues()
		// TODO (b5): this is validation logic, should happen before WriteDataset is ever called
		// all resources must be references
		for key, r := range ds.Transform.Resources {
			if r.Path == "" {
				return nil, fmt.Errorf("transform resource %s requires a path to save", key)
			}
		}

		tff, err := JSONFile(PackageFileTransform.Filename(), ds.Transform)
		if err != nil {
			return nil, err
		}

		if tfsf := ds.Transform.ScriptFile(); tfsf != nil {
			files = append(files, qfs.NewMemfileReader(transformScriptFilename, tfsf))

			hook := func(ctx context.Context, f qfs.File, pathMap map[string]string) (io.Reader, error) {
				ds.Transform.ScriptPath = pathMap[transformScriptFilename]
				return JSONFile(PackageFileTransform.Filename(), ds.Transform)
			}
			tff = qfs.NewWriteHookFile(tff, hook, transformScriptFilename)
		}

		files = append(files, tff)
		components = append(components, PackageFileTransform.Filename())
	}

	if ds.Readme != nil {
		ds.Readme.DropTransientValues()

		rmf, err := JSONFile(PackageFileReadme.Filename(), ds.Transform)
		if err != nil {
			return nil, err
		}

		if rmsf := ds.Readme.ScriptFile(); rmsf != nil {
			files = append(files, qfs.NewMemfileReader(PackageFileReadmeScript.Filename(), rmsf))
			hook := func(ctx context.Context, f qfs.File, pathMap map[string]string) (io.Reader, error) {
				ds.Readme.ScriptPath = pathMap[PackageFileReadmeScript.Filename()]
				return JSONFile(PackageFileReadme.Filename(), ds.Transform)
			}
			rmf = qfs.NewWriteHookFile(rmf, hook, PackageFileReadmeScript.Filename())
		}

		files = append(files, rmf)
		components = append(components, PackageFileReadme.Filename())
	}

	if ds.Viz != nil {
		ds.Viz.DropTransientValues()
		deps := []string{}

		vzfs := ds.Viz.ScriptFile()
		if vzfs != nil {
			files = append(files, qfs.NewMemfileReader(PackageFileVizScript.Filename(), vzfs))
			deps = append(deps, PackageFileVizScript.Filename())
		}

		renderedF := ds.Viz.RenderedFile()
		if renderedF != nil {
			files = append(files, qfs.NewMemfileReader(PackageFileRenderedViz.Filename(), renderedF))
			deps = append(deps, PackageFileRenderedViz.Filename())
		} else if vzfs != nil && sw.ShouldRender {
			deps = append(deps, PackageFileRenderedViz.Filename())
			hook := renderVizWriteHook(fs, ds, bodyFullPath)
			renderedF = qfs.NewWriteHookFile(emptyFile(PackageFileRenderedViz.Filename()), hook, append([]string{PackageFileVizScript.Filename()}, components...)...)
			files = append(files, renderedF)
		}

		vzf, err := JSONFile(PackageFileViz.Filename(), ds.Viz)
		if err != nil {
			return nil, err
		}
		hook := func(ctx context.Context, f qfs.File, added map[string]string) (io.Reader, error) {
			for _, dep := range deps {
				switch dep {
				case PackageFileVizScript.Filename():
					ds.Viz.ScriptPath = added[dep]
				case PackageFileRenderedViz.Filename():
					ds.Viz.RenderedPath = added[dep]
				}
			}
			return JSONFile(PackageFileViz.Filename(), ds.Viz)
		}

		vzf = qfs.NewWriteHookFile(vzf, hook, deps...)
	}

	if ds.Commit != nil {
		hook := jsonWriteHook(PackageFileCommit.Filename(), ds.Commit)
		cmf := qfs.NewWriteHookFile(emptyFile(PackageFileCommit.Filename()), hook, components...)
		files = append(files, cmf)
		components = append(components, PackageFileCommit.Filename())
	}

	hook := func(ctx context.Context, f qfs.File, pathMap map[string]string) (io.Reader, error) {
		log.Debugf("writing dataset file. components=%v", components)
		for _, comp := range components {
			switch comp {
			case PackageFileCommit.Filename():
				ds.Commit = dataset.NewCommitRef(pathMap[comp])
			case PackageFileReadme.Filename():
				ds.Readme = dataset.NewReadmeRef(pathMap[comp])
			case PackageFileStructure.Filename():
				ds.Structure = dataset.NewStructureRef(pathMap[comp])
			case PackageFileViz.Filename():
				ds.Viz = dataset.NewVizRef(pathMap[comp])
			case PackageFileMeta.Filename():
				ds.Meta = dataset.NewMetaRef(pathMap[comp])
			case bodyFullPath:
				ds.BodyPath = pathMap[comp]
			}
		}
		return JSONFile(PackageFileDataset.Filename(), ds)
	}
	log.Debugf("writing dataset with components=%v", components)
	dsf := qfs.NewWriteHookFile(qfs.NewMemfileBytes(PackageFileDataset.Filename(), []byte{}), hook, components...)
	files = append(files, dsf)

	return qfs.NewMemdir("/", files...), nil
}

func emptyFile(fullPath string) qfs.File {
	return qfs.NewMemfileBytes(fullPath, []byte{})
}

func jsonWriteHook(filename string, data json.Marshaler) qfs.WriteHook {
	return func(ctx context.Context, f qfs.File, pathMap map[string]string) (io.Reader, error) {
		return JSONFile(filename, data)
	}
}
