package dsfs

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multihash"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/dsviz"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/deepdiff"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base/friendly"
	"github.com/qri-io/qri/base/toqtype"
)

// LoadDataset reads a dataset from a cafs and dereferences structure, transform, and commitMsg if they exist,
// returning a fully-hydrated dataset
func LoadDataset(ctx context.Context, store cafs.Filestore, path string) (*dataset.Dataset, error) {
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

// CreateDataset places a new dataset in the store. Admittedly, this isn't a simple process.
// Store is where we're going to
// Dataset to be saved
// Pin the dataset if the underlying store supports the pinning interface
// All streaming files (Body, Transform Script, Viz Script) Must be Resolved before calling if data their data is to be saved
func CreateDataset(ctx context.Context, store cafs.Filestore, ds, dsPrev *dataset.Dataset, pk crypto.PrivKey, pin, force, shouldRender bool) (path string, err error) {

	if pk == nil {
		err = fmt.Errorf("private key is required to create a dataset")
		return
	}
	if err = DerefDataset(ctx, store, ds); err != nil {
		log.Debug(err.Error())
		return
	}
	if err = validate.Dataset(ds); err != nil {
		log.Debug(err.Error())
		return
	}

	if dsPrev != nil && !dsPrev.IsEmpty() {
		if err = DerefDataset(ctx, store, dsPrev); err != nil {
			log.Debug(err.Error())
			return
		}
		if err = validate.Dataset(dsPrev); err != nil {
			log.Debug(err.Error())
			return
		}
	}
	err = prepareDataset(store, ds, dsPrev, pk, force, shouldRender)
	if err != nil {
		log.Debug(err.Error())
		return
	}

	path, err = WriteDataset(ctx, store, ds, pin)
	if err != nil {
		log.Debug(err.Error())
		err = fmt.Errorf("error writing dataset: %s", err.Error())
	}
	return
}

// Timestamp is an function for getting commit timestamps
// timestamps MUST be stored in UTC time zone
var Timestamp = func() time.Time {
	return time.Now().UTC()
}

// prepareDataset modifies a dataset in preparation for adding to a dsfs
// it returns a new data file for use in WriteDataset
func prepareDataset(store cafs.Filestore, ds, dsPrev *dataset.Dataset, privKey crypto.PrivKey, force, shouldRender bool) error {
	var (
		err error
		// lock for parallel edits to ds pointer
		mu sync.Mutex
		// accumulate reader into a buffer for shasum calculation & passing out another qfs.File
		buf    bytes.Buffer
		bf     = ds.BodyFile()
		bfPrev qfs.File
	)

	if dsPrev != nil {
		bfPrev = dsPrev.BodyFile()
	}

	if bf == nil && bfPrev == nil {
		return fmt.Errorf("bodyfile or previous bodyfile needed")
	}

	// If bf is nil, this is a metadata update and we will assume that the
	// previous version of the file had passed validation.
	if bf == nil {
		bf = bfPrev
	} else {
		errR, errW := io.Pipe()
		entryR, entryW := io.Pipe()
		hashR, hashW := io.Pipe()
		done := make(chan error)
		tasks := 3
		valChan := make(chan []jsonschema.ValError)

		go setErrCount(ds, qfs.NewMemfileReader(bf.FileName(), errR), &mu, done, valChan)
		go setDepthAndEntryCount(ds, qfs.NewMemfileReader(bf.FileName(), entryR), &mu, done)
		go setChecksumAndLength(ds, qfs.NewMemfileReader(bf.FileName(), hashR), &buf, &mu, done)

		go func() {
			// Manually closed in order to correctly trigger EOF when reading
			// from the pipes.
			defer errW.Close()
			defer entryW.Close()
			defer hashW.Close()

			// Use a MultiWriter to dispatch the body file to all three
			// routines.
			mw := io.MultiWriter(errW, entryW, hashW)
			io.Copy(mw, bf)
		}()

		// Get validation errors from the setErrCount goroutine.
		// TODO: does this mean that the next block of code only needs to wait
		//       for the other two routines to finish? I.E. can we get results
		//       out of valChan without a matching value in `done`?
		var validationErrors []jsonschema.ValError
		validationErrors = <-valChan

		// Join the outstanding tasks, waiting until all are complete.
		for i := 0; i < tasks; i++ {
			if err := <-done; err != nil {
				return err
			}
		}

		// If in strict mode, fail if there were any errors.
		if ds.Structure.Strict && ds.Structure.ErrCount > 0 {
			fmt.Fprintf(os.Stderr, "\nShowing errors at each /row/column of the dataset body:\n")
			for i, v := range validationErrors {
				fmt.Fprintf(os.Stderr, "%d) %v\n", i, v)
			}
			return fmt.Errorf("strict mode: dataset body did not validate against its schema")
		}
	}

	if err = generateCommit(dsPrev, ds, privKey, force); err != nil {
		return err
	}

	ds.SetBodyFile(qfs.NewMemfileBytes("body."+ds.Structure.Format, buf.Bytes()))

	if shouldRender && ds.Viz != nil && ds.Viz.ScriptFile() != nil {
		// render the viz
		renderedFile, err := dsviz.Render(ds)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error rendering visualization: %s", err.Error())
		}
		ds.Viz.SetRenderedFile(renderedFile)
	}

	return nil
}

// generateCommit creates the commit title, message, timestamp, etc
func generateCommit(prev, ds *dataset.Dataset, privKey crypto.PrivKey, force bool) error {
	shortTitle, longMessage, err := generateCommitDescriptions(prev, ds, force)
	if err != nil {
		log.Debug(fmt.Errorf("error saving: %s", err))
		return fmt.Errorf("error saving: %s", err)
	}

	if ds.Commit.Title == "" {
		ds.Commit.Title = shortTitle
	}
	if ds.Commit.Message == "" {
		ds.Commit.Message = longMessage
	}

	ds.Commit.Timestamp = Timestamp()
	sb, _ := ds.SignableBytes()
	signedBytes, err := privKey.Sign(sb)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error signing commit title: %s", err.Error())
	}
	ds.Commit.Signature = base64.StdEncoding.EncodeToString(signedBytes)

	return nil
}

// setErrCount consumes sets the ErrCount field of a dataset's Structure
func setErrCount(ds *dataset.Dataset, data qfs.File, mu *sync.Mutex, done chan error, valChan chan []jsonschema.ValError) {
	defer data.Close()

	er, err := dsio.NewEntryReader(ds.Structure, data)
	if err != nil {
		log.Debug(err.Error())
		valChan <- nil
		done <- fmt.Errorf("reading data values: %s", err.Error())
		return
	}

	// Send validation errors immediately, before main thread blocks.
	validationErrors, err := validate.EntryReader(er)
	valChan <- validationErrors

	if err != nil {
		log.Debug(err.Error())
		done <- fmt.Errorf("validating data: %s", err.Error())
		return
	}

	mu.Lock()
	ds.Structure.ErrCount = len(validationErrors)
	mu.Unlock()

	done <- nil
}

// setDepthAndEntryCount set the Entries field of a ds.Structure
func setDepthAndEntryCount(ds *dataset.Dataset, data qfs.File, mu *sync.Mutex, done chan error) {
	defer data.Close()
	er, err := dsio.NewEntryReader(ds.Structure, data)
	if err != nil {
		log.Debug(err.Error())
		done <- fmt.Errorf("error reading data values: %s", err.Error())
		return
	}

	entries := 0

	depth := 0
	var ent dsio.Entry
	for {
		if ent, err = er.ReadEntry(); err != nil {
			log.Debug(err.Error())
			break
		}
		// get the depth of this entry, update depth if larger
		if d := getDepth(ent.Value); d > depth {
			depth = d
		}
		entries++
	}
	if err.Error() != "EOF" {
		done <- fmt.Errorf("error reading values at entry %d: %s", entries, err.Error())
		return
	}

	mu.Lock()
	ds.Structure.Entries = entries
	ds.Structure.Depth = depth + 1 // need to add one for the original enclosure
	mu.Unlock()

	done <- nil
}

// getDepth finds the deepest value in a given interface value
func getDepth(x interface{}) (depth int) {
	switch v := x.(type) {
	case map[string]interface{}:
		for _, el := range v {
			if d := getDepth(el); d > depth {
				depth = d
			}
		}
		return depth + 1
	case []interface{}:
		for _, el := range v {
			if d := getDepth(el); d > depth {
				depth = d
			}
		}
		return depth + 1
	default:
		return depth
	}
}

// setChecksumAndLength
func setChecksumAndLength(ds *dataset.Dataset, data qfs.File, buf *bytes.Buffer, mu *sync.Mutex, done chan error) {
	defer data.Close()

	if _, err := io.Copy(buf, data); err != nil {
		done <- err
		return
	}

	shasum, err := multihash.Sum(buf.Bytes(), multihash.SHA2_256, -1)
	if err != nil {
		log.Debug(err.Error())
		done <- fmt.Errorf("error calculating hash: %s", err.Error())
		return
	}

	mu.Lock()
	ds.Structure.Checksum = shasum.B58String()
	ds.Structure.Length = len(buf.Bytes())
	mu.Unlock()

	done <- nil
}

// returns a commit message based on the diff of the two datasets
func generateCommitDescriptions(prev, ds *dataset.Dataset, force bool) (short, long string, err error) {
	if prev == nil || prev.IsEmpty() {
		return "created dataset", "created dataset", nil
	}

	// TODO(dlong): Inline body if it is a reasonable size, in order to get information about
	// how the body has changed.
	// TODO(dlong): Also should ignore derived fields, like structure.{checksum,entries,length}.

	var prevData map[string]interface{}
	prevData, err = toqtype.StructToMap(prev)
	if err != nil {
		return "", "", err
	}

	var nextData map[string]interface{}
	nextData, err = toqtype.StructToMap(ds)
	if err != nil {
		return "", "", err
	}

	stat := deepdiff.Stats{}
	diff, err := deepdiff.Diff(prevData, nextData, deepdiff.OptionSetStats(&stat))
	if err != nil {
		return "", "", err
	}

	shortTitle, longMessage := friendly.DiffDescriptions(diff, &stat)
	if shortTitle == "" {
		if force {
			return "forced update", "forced update", nil
		}
		return "", "", fmt.Errorf("no changes")
	}

	return shortTitle, longMessage, nil
}

// WriteDataset writes a dataset to a cafs, replacing subcomponents of a dataset with path references
// during the write process. Directory structure is according to PackageFile naming conventions.
// This method is currently exported, but 99% of use cases should use CreateDataset instead of this
// lower-level function
func WriteDataset(ctx context.Context, store cafs.Filestore, ds *dataset.Dataset, pin bool) (string, error) {

	if ds == nil || ds.IsEmpty() {
		return "", fmt.Errorf("cannot save empty dataset")
	}
	name := ds.Name // preserve name for body file
	bodyFile := ds.BodyFile()
	fileTasks := 0
	addedDataset := false
	adder, err := store.NewAdder(pin, true)
	if err != nil {
		return "", fmt.Errorf("error creating new adder: %s", err.Error())
	}

	if ds.Viz != nil {
		ds.Viz.DropTransientValues()
		vizScript := ds.Viz.ScriptFile()
		vizRendered := ds.Viz.RenderedFile()
		// add task for the viz.json
		if vizRendered != nil {
			// add the rendered visualization
			// and add working group for adding the viz script file
			vrFile := qfs.NewMemfileReader(PackageFileRenderedViz.String(), vizRendered)
			defer vrFile.Close()
			fileTasks++
			adder.AddFile(ctx, vrFile)
		} else if vizScript != nil {
			// add the vizScript
			vsFile := qfs.NewMemfileReader(vizScriptFilename, vizScript)
			defer vsFile.Close()
			fileTasks++
			adder.AddFile(ctx, vsFile)
		} else {
			vizdata, err := json.Marshal(ds.Viz)
			if err != nil {
				return "", fmt.Errorf("error marshalling dataset viz to json: %s", err.Error())
			}
			fileTasks++
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
			fileTasks++
			adder.AddFile(ctx, rmFile)
		} else if readmeScript != nil {
			// add the readmeScript
			rmFile := qfs.NewMemfileReader(readmeScriptFilename, readmeScript)
			defer rmFile.Close()
			fileTasks++
			adder.AddFile(ctx, rmFile)
		} else {
			readmeData, err := json.Marshal(ds.Readme)
			if err != nil {
				return "", fmt.Errorf("error marshalling dataset readme to json: %s", err.Error())
			}
			fileTasks++
			adder.AddFile(ctx, qfs.NewMemfileBytes(PackageFileReadme.String(), readmeData))
		}
	}

	if ds.Meta != nil {
		mdf, err := JSONFile(PackageFileMeta.String(), ds.Meta)
		if err != nil {
			return "", fmt.Errorf("error marshaling metadata to json: %s", err.Error())
		}
		fileTasks++
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
			fileTasks++
			adder.AddFile(ctx, tsFile)
			// NOTE - add wg for the transform.json file ahead of time, which isn't completed
			// until after scriptPath has been added
		} else {
			tfdata, err := json.Marshal(ds.Transform)
			if err != nil {
				return "", fmt.Errorf("error marshalling dataset transform to json: %s", err.Error())
			}

			fileTasks++
			adder.AddFile(ctx, qfs.NewMemfileBytes(PackageFileTransform.String(), tfdata))
		}
	}

	if ds.Commit != nil {
		ds.Commit.DropTransientValues()
		cmf, err := JSONFile(PackageFileCommit.String(), ds.Commit)
		if err != nil {
			return "", fmt.Errorf("error marshilng dataset commit message to json: %s", err.Error())
		}
		fileTasks++
		adder.AddFile(ctx, cmf)
	}

	if ds.Structure != nil {
		ds.Structure.DropTransientValues()
		stf, err := JSONFile(PackageFileStructure.String(), ds.Structure)
		if err != nil {
			return "", fmt.Errorf("error marshaling dataset structure to json: %s", err.Error())
		}
		fileTasks++
		adder.AddFile(ctx, stf)
	}

	fileTasks++
	adder.AddFile(ctx, bodyFile)

	var path string
	done := make(chan error, 0)
	go func() {
		for ao := range adder.Added() {
			path = ao.Path
			switch ao.Name {
			case PackageFileStructure.String():
				ds.Structure = dataset.NewStructureRef(ao.Path)
			case PackageFileTransform.String():
				ds.Transform = dataset.NewTransformRef(ao.Path)
			case PackageFileMeta.String():
				ds.Meta = dataset.NewMetaRef(ao.Path)
			case PackageFileCommit.String():
				ds.Commit = dataset.NewCommitRef(ao.Path)
			case PackageFileViz.String():
				ds.Viz = dataset.NewVizRef(ao.Path)
			case bodyFile.FileName():
				ds.BodyPath = ao.Path
				// ds.SetBodyFile(qfs.NewMemfileBytes(bodyFile.FileName(), bodyBytesBuf.Bytes()))
			case transformScriptFilename:
				ds.Transform.ScriptPath = ao.Path
				tfdata, err := json.Marshal(ds.Transform)
				if err != nil {
					done <- err
					return
				}
				// Add the encoded transform file, decrementing the stray fileTasks from above
				fileTasks++
				adder.AddFile(ctx, qfs.NewMemfileBytes(PackageFileTransform.String(), tfdata))
			case PackageFileRenderedViz.String():
				ds.Viz.RenderedPath = ao.Path
				vsFile := qfs.NewMemfileReader(vizScriptFilename, ds.Viz.ScriptFile())
				defer vsFile.Close()
				fileTasks++
				adder.AddFile(ctx, vsFile)
			case vizScriptFilename:
				ds.Viz.ScriptPath = ao.Path
				vizdata, err := json.Marshal(ds.Viz)
				if err != nil {
					done <- err
					return
				}
				// Add the encoded transform file, decrementing the stray fileTasks from above
				fileTasks++
				adder.AddFile(ctx, qfs.NewMemfileBytes(PackageFileViz.String(), vizdata))
			case PackageFileRenderedReadme.String():
				ds.Readme.RenderedPath = ao.Path
				vsFile := qfs.NewMemfileReader(readmeScriptFilename, ds.Readme.ScriptFile())
				defer vsFile.Close()
				fileTasks++
				adder.AddFile(ctx, vsFile)
			case readmeScriptFilename:
				ds.Readme.ScriptPath = ao.Path
				readmeData, err := json.Marshal(ds.Readme)
				if err != nil {
					done <- err
					return
				}
				// Add the encoded transform file, decrementing the stray fileTasks from above
				fileTasks++
				adder.AddFile(ctx, qfs.NewMemfileBytes(PackageFileReadme.String(), readmeData))
			}

			fileTasks--
			if fileTasks == 0 {
				if !addedDataset {
					ds.DropTransientValues()
					dsdata, err := json.Marshal(ds)
					if err != nil {
						done <- err
						return
					}

					adder.AddFile(ctx, qfs.NewMemfileBytes(PackageFileDataset.String(), dsdata))
				}
				//
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
		return path, err
	}
	// TODO (b5): currently we're loading to keep the ds pointer hydrated post-write
	// we should remove that assumption, allowing callers to skip this load step, which may
	// be unnecessary
	var loaded *dataset.Dataset
	loaded, err = LoadDataset(ctx, store, path)
	if err != nil {
		return "", err
	}
	loaded.Name = name
	*ds = *loaded
	return path, nil
}
