package dsfs

import (
	"context"
	"encoding/base64"
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

	// assign path to retain internal reference to the
	// path this dataset was read from
	ds.Assign(dataset.NewDatasetRef(path))

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
		log.Debugf("dereferencing dataset components: %s", err)
		return "", err
	}
	if err := validate.Dataset(ds); err != nil {
		log.Debug(err.Error())
		return "", err
	}
	log.Debugf("CreateDataset ds.Peername=%q ds.Name=%q writeDestType=%s", ds.Peername, ds.Name, destination.Type())

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
	root, err := buildFileGraph(destination, ds, pk, sw)
	if err != nil {
		return "", err
	}

	return qfs.WriteWithHooks(ctx, destination, root)
}

func buildFileGraph(fs qfs.Filesystem, ds *dataset.Dataset, privKey crypto.PrivKey, sw SaveSwitches) (root qfs.File, err error) {
	var (
		files               []qfs.File
		bdf                 = ds.BodyFile()
		packageBodyFilepath string
	)

	if bdf != nil {
		files = append(files, bdf)
		packageBodyFilepath = bdf.FullPath()
	}

	if ds.Structure != nil {
		ds.Structure.DropTransientValues()

		stf, err := JSONFile(PackageFileStructure.Filename(), ds.Structure)
		if err != nil {
			return nil, err
		}

		if bdf != nil {
			hook := func(ctx context.Context, f qfs.File, added map[string]string) (io.Reader, error) {
				if processingFile, ok := bdf.(doneProcessingFile); ok {
					err := <-processingFile.DoneProcessing()
					if err != nil {
						return nil, err
					}
				}
				return JSONFile(f.FullPath(), ds.Structure)
			}
			stf = qfs.NewWriteHookFile(stf, hook, packageBodyFilepath)
		}

		files = append(files, stf)
	}

	// stats relies on a structure component & a body file
	if statsCompFile, ok := bdf.(statsComponentFile); ok {
		hook := func(ctx context.Context, f qfs.File, added map[string]string) (io.Reader, error) {
			sa, err := statsCompFile.StatsComponent()
			if err != nil {
				return nil, err
			}
			ds.Stats = sa
			return JSONFile(f.FullPath(), sa)
		}

		hookFile := qfs.NewWriteHookFile(qfs.NewMemfileBytes(PackageFileStats.Filename(), []byte{}), hook, PackageFileStructure.Filename())
		files = append(files, hookFile)
	}

	if ds.Meta != nil {
		ds.Meta.DropTransientValues()
		mdf, err := JSONFile(PackageFileMeta.Filename(), ds.Meta)
		if err != nil {
			return nil, fmt.Errorf("encoding meta component to json: %w", err)
		}
		files = append(files, mdf)
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
	}

	if ds.Readme != nil {
		ds.Readme.DropTransientValues()

		if rmsf := ds.Readme.ScriptFile(); rmsf != nil {
			files = append(files, qfs.NewMemfileReader(PackageFileReadmeScript.Filename(), rmsf))
		}
	}

	if ds.Viz != nil {
		ds.Viz.DropTransientValues()

		vzfs := ds.Viz.ScriptFile()
		if vzfs != nil {
			files = append(files, qfs.NewMemfileReader(PackageFileVizScript.Filename(), vzfs))
		}

		renderedF := ds.Viz.RenderedFile()
		if renderedF != nil {
			files = append(files, qfs.NewMemfileReader(PackageFileRenderedViz.Filename(), renderedF))
		} else if vzfs != nil && sw.ShouldRender {
			hook := renderVizWriteHook(fs, ds, packageBodyFilepath)
			renderedF = qfs.NewWriteHookFile(emptyFile(PackageFileRenderedViz.Filename()), hook, append([]string{PackageFileVizScript.Filename()}, filePaths(files)...)...)
			files = append(files, renderedF)
		}

		// we don't add the viz component itself, it's inlined in dataset.json
	}

	if ds.Commit != nil {
		hook := func(ctx context.Context, f qfs.File, added map[string]string) (io.Reader, error) {

			for filepath, addr := range added {
				switch filepath {
				case PackageFileVizScript.Filename():
					ds.Viz.ScriptPath = addr
				case PackageFileRenderedViz.Filename():
					ds.Viz.RenderedPath = addr
				case PackageFileReadmeScript.Filename():
					ds.Readme.ScriptPath = addr
				case PackageFileStructure.Filename():
					ds.Structure = dataset.NewStructureRef(addr)
				case PackageFileViz.Filename():
					ds.Viz = dataset.NewVizRef(addr)
				case PackageFileMeta.Filename():
					ds.Meta = dataset.NewMetaRef(addr)
				case PackageFileStats.Filename():
					ds.Stats = dataset.NewStatsRef(addr)
				case packageBodyFilepath:
					ds.BodyPath = addr
				}
			}

			signedBytes, err := privKey.Sign(ds.SigningBytes())
			if err != nil {
				log.Debug(err.Error())
				return nil, fmt.Errorf("error signing commit title: %s", err.Error())
			}
			ds.Commit.Signature = base64.StdEncoding.EncodeToString(signedBytes)
			return JSONFile(PackageFileCommit.Filename(), ds.Commit)
		}

		cmf := qfs.NewWriteHookFile(emptyFile(PackageFileCommit.Filename()), hook, filePaths(files)...)
		files = append(files, cmf)
	}

	pkgFiles := filePaths(files)
	if len(pkgFiles) == 0 {
		return nil, fmt.Errorf("cannot save empty dataset")
	}

	hook := func(ctx context.Context, f qfs.File, pathMap map[string]string) (io.Reader, error) {
		log.Debugf("writing dataset file. files=%v", pkgFiles)
		ds.DropTransientValues()

		for _, comp := range pkgFiles {
			switch comp {
			case PackageFileCommit.Filename():
				ds.Commit = dataset.NewCommitRef(pathMap[comp])
			}
		}
		return JSONFile(PackageFileDataset.Filename(), ds)
	}

	dsf := qfs.NewWriteHookFile(qfs.NewMemfileBytes(PackageFileDataset.Filename(), []byte{}), hook, filePaths(files)...)
	files = append(files, dsf)

	log.Debugf("constructing dataset with pkgFiles=%v", pkgFiles)
	return qfs.NewMemdir("/", files...), nil
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

func jsonWriteHook(filename string, data json.Marshaler) qfs.WriteHook {
	return func(ctx context.Context, f qfs.File, pathMap map[string]string) (io.Reader, error) {
		return JSONFile(filename, data)
	}
}
