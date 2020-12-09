package dsfs

import (
	"context"
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
)

// ErrStrictMode indicates a dataset failed validation when it is required to
// pass (Structure.Strict == true)
var ErrStrictMode = fmt.Errorf("dataset body did not validate against schema in strict-mode")

// DerefStructure derferences a dataset's structure element if required
// should be a no-op if ds.Structure is nil or isn't a reference
func DerefStructure(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Structure != nil && ds.Structure.IsEmpty() && ds.Structure.Path != "" {
		st, err := loadStructure(ctx, store, ds.Structure.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("loading dataset structure: %w", err)
		}
		// assign path to retain internal reference to path
		st.Path = ds.Structure.Path
		ds.Structure = st
	}
	return nil
}

// loadStructure assumes path is valid
func loadStructure(ctx context.Context, fs qfs.Filesystem, path string) (st *dataset.Structure, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading structure file: %s", err.Error())
	}
	return dataset.UnmarshalStructure(data)
}

func structureFileAddFunc(destFS qfs.Filesystem) addWriteFileFunc {
	return func(ds *dataset.Dataset, wfs *writeFiles) (err error) {
		if ds.Structure == nil {
			return nil
		}

		ds.Structure.DropTransientValues()

		if wfs.body == nil {
			log.Debugf("body is nil, using json structure file")
			wfs.structure, err = JSONFile(PackageFileStructure.Filename(), ds.Structure)
			return err
		}

		hook := func(ctx context.Context, f qfs.File, added map[string]string) (io.Reader, error) {
			if processingFile, ok := wfs.body.(doneProcessingFile); ok {
				if err := <-processingFile.DoneProcessing(); err != nil {
					return nil, err
				}
			}

			// if the destination filesystem is content-addressed, use the body
			// path as the checksum. Include path prefix to disambiguate which FS
			// generated the checksum
			if _, ok := destFS.(qfs.CAFS); ok {
				if path, ok := added[wfs.body.FullPath()]; ok {
					ds.Structure.Checksum = path
				}
			}

			return JSONFile(f.FullPath(), ds.Structure)
		}

		wfs.structure = qfs.NewWriteHookFile(emptyFile(PackageFileStructure.Filename()), hook, wfs.body.FullPath())
		return nil
	}
}
