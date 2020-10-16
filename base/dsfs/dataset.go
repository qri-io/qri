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
	"github.com/qri-io/qfs/cafs"
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
func LoadDataset(ctx context.Context, store cafs.Filestore, path string) (*dataset.Dataset, error) {
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
func LoadDatasetRefs(ctx context.Context, store cafs.Filestore, path string) (*dataset.Dataset, error) {
	log.Debugf("LoadDatasetRefs path=%q", path)
	ds := dataset.NewDatasetRef(path)

	pathWithBasename := PackageFilepath(store, path, PackageFileDataset)
	data, err := fileBytes(store.Get(ctx, pathWithBasename))
	// if err != nil {
	// 	return nil, fmt.Errorf("error getting file bytes: %s", err.Error())
	// }

	// TODO - for some reason files are sometimes coming back empty from IPFS,
	// every now & then. In the meantime, let's give a second try if data is empty
	if err != nil || len(data) == 0 {
		data, err = fileBytes(store.Get(ctx, pathWithBasename))
		if err != nil {
			log.Debug(err.Error())
			return nil, fmt.Errorf("error getting file bytes: %s", err.Error())
		}
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
func DerefDataset(ctx context.Context, store cafs.Filestore, ds *dataset.Dataset) error {
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
func DerefDatasetStructure(ctx context.Context, store cafs.Filestore, ds *dataset.Dataset) error {
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
func DerefDatasetViz(ctx context.Context, store cafs.Filestore, ds *dataset.Dataset) error {
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
func DerefDatasetReadme(ctx context.Context, store cafs.Filestore, ds *dataset.Dataset) error {
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
func DerefDatasetTransform(ctx context.Context, store cafs.Filestore, ds *dataset.Dataset) error {
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
func DerefDatasetMeta(ctx context.Context, store cafs.Filestore, ds *dataset.Dataset) error {
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
func DerefDatasetCommit(ctx context.Context, store cafs.Filestore, ds *dataset.Dataset) error {
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
	source, destination cafs.Filestore,
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

	var hooks []*MerkelizeHook

	// TODO (b5) - set renderedFile to a zero-length (but non-nil) file so the count
	// in WriteDataset is correct
	if sw.ShouldRender && ds.Viz != nil && ds.Viz.ScriptFile() != nil && ds.Viz.RenderedFile() == nil {
		ds.Viz.SetRenderedFile(qfs.NewMemfileBytes(PackageFileRenderedViz.String(), []byte{}))
		hooks = append(hooks, renderVizMerkleHook(dsLk, ds, bodyFile.FileName()))
	}

	path, err := WriteDataset(ctx, dsLk, destination, ds, pk, sw.Pin, hooks...)
	if err != nil {
		log.Debug(err.Error())
		return "", err
	}
	return path, nil
}

func concatFunc(f1, f2 func()) func() {
	return func() {
		f1()
		f2()
	}
}

// MerkelizeCallback is a function that's called when a given path has been
// written to the content addressed filesystem
type MerkelizeCallback func(ctx context.Context, store cafs.Filestore, merkelizedPaths map[string]string) (io.Reader, error)

// MerkelizeHook configures a callback function to be executed on a saved
// file, at a specific point in the merkelization process
type MerkelizeHook struct {
	// path of file to fire on
	inputFilename string
	path          string
	once          sync.Once
	// slice of pre-merkelized paths that need to be saved before the hook
	// can be called
	requiredPaths []string
	// function to call
	callback MerkelizeCallback
}

// NewMerkelizeHook creates
func NewMerkelizeHook(inputFilename string, cb MerkelizeCallback, requiredPaths ...string) *MerkelizeHook {
	return &MerkelizeHook{
		inputFilename: inputFilename,
		requiredPaths: requiredPaths,
		callback:      cb,
	}
}

func (h *MerkelizeHook) hasRequiredPaths(merkelizedPaths map[string]string) bool {
	for _, p := range h.requiredPaths {
		if _, ok := merkelizedPaths[p]; !ok {
			return false
		}
	}
	return true
}

// WriteDataset writes a dataset to a cafs, replacing subcomponents of a dataset with path references
// during the write process. Directory structure is according to PackageFile naming conventions.
// This method is currently exported, but 99% of use cases should use CreateDataset instead of this
// lower-level function
func WriteDataset(
	ctx context.Context,
	dsLk *sync.Mutex,
	destination cafs.Filestore,
	ds *dataset.Dataset,
	pk crypto.PrivKey,
	pin bool,
	hooks ...*MerkelizeHook,
) (string, error) {
	if ds == nil || ds.IsEmpty() {
		return "", fmt.Errorf("cannot save empty dataset")
	}

	var rollback = func() {
		log.Debug("rolling back failed write operation")
	}
	defer func() {
		if rollback != nil {
			log.Debug("InitDataset rolling back...")
			rollback()
		}
	}()

	var (
		bodyFile  = ds.BodyFile()
		fileTasks = fileTaskCount(ds)
	)

	log.Debugf("WriteDataset tasks=%d", fileTasks)
	adder, err := destination.NewAdder(pin, true)
	if err != nil {
		return "", fmt.Errorf("creating new CAFS adder: %w", err)
	}

	ds.Peername = ""
	ds.Name = ""

	if ds.Meta != nil {
		mdf, err := JSONFile(PackageFileMeta.String(), ds.Meta)
		if err != nil {
			return "", fmt.Errorf("encoding meta component to json: %w", err)
		}
		adder.AddFile(ctx, mdf)
	}

	if ds.Transform != nil {
		// TODO (b5): this is validation logic, should happen before WriteDataset is ever called
		// all resources must be references
		for key, r := range ds.Transform.Resources {
			if r.Path == "" {
				return "", fmt.Errorf("transform resource %s requires a path to save", key)
			}
		}

		sr := ds.Transform.ScriptFile()
		ds.Transform.DropTransientValues()
		if sr != nil {
			tsFile := qfs.NewMemfileReader(transformScriptFilename, sr)
			defer tsFile.Close()
			adder.AddFile(ctx, tsFile)
		} else {
			tfdata, err := json.Marshal(ds.Transform)
			if err != nil {
				return "", fmt.Errorf("encoding transform component to json: %w", err)
			}
			adder.AddFile(ctx, qfs.NewMemfileBytes(PackageFileTransform.String(), tfdata))
		}
	}

	adder.AddFile(ctx, bodyFile)

	var finalPath string
	done := make(chan error, 0)
	merkelizedPaths := map[string]string{}

	go func() {
		for ao := range adder.Added() {
			log.Debugf("added name=%s hash=%s", ao.Name, ao.Path)
			path := ao.Path
			merkelizedPaths[ao.Name] = ao.Path
			finalPath = ao.Path
			rollback = concatFunc(func() {
				log.Debugf("removing: %s", path)
				if err := destination.Delete(ctx, path); err != nil {
					log.Debugf("error removing path: %s: %s", path, err)
				}
			}, rollback)

			for i, hook := range hooks {
				if hook.hasRequiredPaths(merkelizedPaths) {
					hook.once.Do(func() {
						log.Debugf("calling merkelizeHook path=%s", hook.inputFilename)
						r, err := hook.callback(ctx, destination, merkelizedPaths)
						if err != nil {
							done <- err
							return
						}
						if err = adder.AddFile(ctx, qfs.NewMemfileReader(hook.inputFilename, r)); err != nil {
							done <- err
							return
						}
					})
					hooks = append(hooks[:i], hooks[i:]...)
				} else {
					log.Debugf("missing required paths for hook path=%s required=%#v merkelized=%#v", hook.inputFilename, hook.requiredPaths, merkelizedPaths)
				}
			}

			switch ao.Name {
			case PackageFileStructure.String():
				log.Debugf("added structure. path=%s", ao.Path)
				dsLk.Lock()
				ds.Structure = dataset.NewStructureRef(ao.Path)
				dsLk.Unlock()
			case PackageFileTransform.String():
				log.Debugf("added transform. path=%s", ao.Path)
				ds.Transform = dataset.NewTransformRef(ao.Path)
			case PackageFileMeta.String():
				log.Debugf("added meta. path=%s", ao.Path)
				ds.Meta = dataset.NewMetaRef(ao.Path)
			case PackageFileCommit.String():
				log.Debugf("added commit. path=%s", ao.Path)
				ds.Commit = dataset.NewCommitRef(ao.Path)
			case PackageFileViz.String():
				log.Debugf("added viz. path=%s", ao.Path)
				ds.Viz = dataset.NewVizRef(ao.Path)
			case bodyFile.FileName():
				log.Debugf("added body. path=%s", ao.Path)

				if dpf, ok := bodyFile.(doneProcessingFile); ok {
					if err := <-dpf.DoneProcessing(); err != nil {
						done <- err
						return
					}
				}

				dsLk.Lock()
				ds.BodyPath = ao.Path
				if ds.Structure != nil {
					ds.Structure.DropTransientValues()
					stf, err := JSONFile(PackageFileStructure.String(), ds.Structure)
					if err != nil {
						done <- fmt.Errorf("encoding structure component to json: %w", err)
						return
					}
					adder.AddFile(ctx, stf)
				}

				if ds.Viz != nil {
					ds.Viz.DropTransientValues()
					vizScript := ds.Viz.ScriptFile()

					// vizRendered := ds.Viz.RenderedFile()
					// add task for the viz.json
					// if vizRendered != nil {
					// // add the rendered visualization
					// // and add working group for adding the viz script file
					// vrFile := qfs.NewMemfileReader(PackageFileRenderedViz.String(), vizRendered)
					// defer vrFile.Close()
					// adder.AddFile(ctx, vrFile)

					if vizScript != nil {
						// add the vizScript
						vsFile := qfs.NewMemfileReader(vizScriptFilename, vizScript)
						defer vsFile.Close()
						adder.AddFile(ctx, vsFile)
					} else {
						vizdata, err := json.Marshal(ds.Viz)
						if err != nil {
							done <- fmt.Errorf("encoding viz component to json: %w", err)
							return
						}
						adder.AddFile(ctx, qfs.NewMemfileBytes(PackageFileViz.String(), vizdata))
					}
				}

				if ds.Readme != nil {
					ds.Readme.DropTransientValues()
					readmeScript := ds.Readme.ScriptFile()
					readmeRendered := ds.Readme.RenderedFile()
					// add task for the readme
					if readmeRendered != nil {
						// add the rendered visualization
						// and add working group for adding the viz script file
						rmFile := qfs.NewMemfileReader(PackageFileRenderedReadme.String(), readmeRendered)
						defer rmFile.Close()
						adder.AddFile(ctx, rmFile)
					} else if readmeScript != nil {
						// add the readmeScript
						rmFile := qfs.NewMemfileReader(PackageFileReadmeScript.String(), readmeScript)
						defer rmFile.Close()
						adder.AddFile(ctx, rmFile)
					} else {
						readmeData, err := json.Marshal(ds.Readme)
						if err != nil {
							done <- fmt.Errorf("encoding readme component to json: %w", err)
							return
						}
						adder.AddFile(ctx, qfs.NewMemfileBytes(PackageFileReadme.String(), readmeData))
					}
				}
				dsLk.Unlock()
			case transformScriptFilename:
				log.Debugf("added transform script. path=%s", ao.Path)
				ds.Transform.ScriptPath = ao.Path
				tfdata, err := json.Marshal(ds.Transform)
				if err != nil {
					done <- err
					return
				}
				adder.AddFile(ctx, qfs.NewMemfileBytes(PackageFileTransform.String(), tfdata))
			case PackageFileRenderedViz.String():
				log.Debugf("added rendered viz. path=%s", ao.Path)
				ds.Viz.RenderedPath = ao.Path
				vsFile := qfs.NewMemfileReader(vizScriptFilename, ds.Viz.ScriptFile())
				defer vsFile.Close()
				adder.AddFile(ctx, vsFile)
			case vizScriptFilename:
				log.Debugf("added viz script. path=%s", ao.Path)
				ds.Viz.ScriptPath = ao.Path
				vizdata, err := json.Marshal(ds.Viz)
				if err != nil {
					done <- err
					return
				}
				// Add the encoded transform file, decrementing the stray fileTasks from above
				adder.AddFile(ctx, qfs.NewMemfileBytes(PackageFileViz.String(), vizdata))
			case PackageFileRenderedReadme.String():
				log.Debugf("added rendered readme. path=%s", ao.Path)
				ds.Readme.RenderedPath = ao.Path
				vsFile := qfs.NewMemfileReader(PackageFileReadmeScript.String(), ds.Readme.ScriptFile())
				defer vsFile.Close()
				adder.AddFile(ctx, vsFile)
			case PackageFileReadmeScript.String():
				log.Debugf("added readme script. path=%s", ao.Path)
				ds.Readme.ScriptPath = ao.Path
				readmeData, err := json.Marshal(ds.Readme)
				if err != nil {
					done <- err
					return
				}
				// Add the encoded transform file, decrementing the stray fileTasks from above
				adder.AddFile(ctx, qfs.NewMemfileBytes(PackageFileReadme.String(), readmeData))
			}

			fileTasks--
			if fileTasks == 1 {
				dsLk.Lock()
				if ds.Commit != nil {
					ds.Commit.DropTransientValues()
					signedBytes, err := pk.Sign(ds.SigningBytes())
					if err != nil {
						log.Debug(err.Error())
						done <- fmt.Errorf("signing commit: %w", err)
						return
					}
					ds.Commit.Signature = base64.StdEncoding.EncodeToString(signedBytes)
					log.Debugf("generateCommit complete. signature=%q", ds.Commit.Signature)
					cmf, err := JSONFile(PackageFileCommit.String(), ds.Commit)
					if err != nil {
						done <- fmt.Errorf("encoding commit component to json: %w", err)
						return
					}
					adder.AddFile(ctx, cmf)
				}
				dsLk.Unlock()
			} else if fileTasks == 0 {
				log.Debug("no fileTasks remain")

				ds.DropTransientValues()
				dsdata, err := json.Marshal(ds)
				if err != nil {
					done <- err
					return
				}

				if addErr := adder.AddFile(ctx, qfs.NewMemfileBytes(PackageFileDataset.String(), dsdata)); addErr != nil {
					log.Debugf("error adding dataset file: %q", addErr)
				}

				if err := adder.Close(); err != nil {
					done <- err
					return
				}
			}
		}
		done <- nil
	}()

	err = <-done
	if err != nil {
		log.Debugf("writing dataset: %q", err)
		return finalPath, err
	}

	log.Debugf("dataset written to filesystem. path=%q", finalPath)

	// TODO(dustmop): This is necessary because ds doesn't have all fields in Structure and Commit.
	// Try if there's another way to set these instead of requiring a full call to LoadDataset.
	var loaded *dataset.Dataset
	loaded, err = LoadDataset(ctx, destination, finalPath)
	if err != nil {
		return "", err
	}
	*ds = *loaded

	// successful execution. remove rollback func
	rollback = nil
	return finalPath, nil
}

// determine the number of files that need to be written by examining which
// dataset components & component file fields are populated
func fileTaskCount(ds *dataset.Dataset) (tasks int) {
	if ds.Commit != nil {
		tasks++
	}

	if ds.Meta != nil {
		tasks++
	}
	if ds.Transform != nil {
		tasks++
		if ds.Transform.ScriptFile() != nil {
			tasks++
		}
	}
	if ds.Viz != nil {
		tasks++
		if ds.Viz.ScriptFile() != nil {
			tasks++
		}
		if ds.Viz.RenderedFile() != nil {
			tasks++
		}
	}
	if ds.Readme != nil {
		tasks++
		if ds.Readme.ScriptFile() != nil {
			tasks++
		}
		if ds.Readme.RenderedFile() != nil {
			tasks++
		}
	}
	if ds.Structure != nil {
		tasks++
	}
	if ds.BodyFile() != nil {
		tasks++
	}
	return tasks
}
