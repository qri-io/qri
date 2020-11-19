package dsfs

import (
	"context"
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
)

// DerefTransform derferences a dataset's transform element if required
// should be a no-op if ds.Structure is nil or isn't a reference
func DerefTransform(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Transform != nil && ds.Transform.IsEmpty() && ds.Transform.Path != "" {
		t, err := loadTransform(ctx, store, ds.Transform.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("loading dataset transform: %w", err)
		}
		t.Path = ds.Transform.Path
		ds.Transform = t
	}
	return nil
}

// loadTransform assumes the provided path is correct
func loadTransform(ctx context.Context, fs qfs.Filesystem, path string) (q *dataset.Transform, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading transform raw data: %s", err.Error())
	}

	return dataset.UnmarshalTransform(data)
}

// ErrNoTransform is the error for asking a dataset without a tranform component for viz info
var ErrNoTransform = fmt.Errorf("this dataset has no transform component")

// LoadTransformScript loads transform script data from a dataset path if the given dataset has a transform script specified
// the returned qfs.File will be the value of dataset.Transform.ScriptPath
// TODO - this is broken, assumes file is JSON. fix & test or depricate
func LoadTransformScript(ctx context.Context, fs qfs.Filesystem, dspath string) (qfs.File, error) {
	ds, err := LoadDataset(ctx, fs, dspath)
	if err != nil {
		return nil, err
	}

	if ds.Transform == nil || ds.Transform.ScriptPath == "" {
		return nil, ErrNoTransform
	}

	return fs.Get(ctx, ds.Transform.ScriptPath)
}

func addTransformFile(ds *dataset.Dataset, wfs *writeFiles) (err error) {
	if ds.Transform == nil {
		return nil
	}

	ds.Transform.DropTransientValues()
	// TODO (b5): this is validation logic, should happen before WriteDataset is ever called
	// all resources must be references
	for key, r := range ds.Transform.Resources {
		if r.Path == "" {
			return fmt.Errorf("transform resource %s requires a path to save", key)
		}
	}

	if tfsf := ds.Transform.ScriptFile(); tfsf != nil {
		wfs.transformScript = qfs.NewMemfileReader(transformScriptFilename, tfsf)
	}

	if wfs.transformScript != nil {
		hook := func(ctx context.Context, f qfs.File, pathMap map[string]string) (io.Reader, error) {
			ds.Transform.ScriptPath = pathMap[transformScriptFilename]
			return JSONFile(PackageFileTransform.Filename(), ds.Transform)
		}
		wfs.transform = qfs.NewWriteHookFile(emptyFile(PackageFileTransform.Filename()), hook, transformScriptFilename)
		return nil
	}

	wfs.transform, err = JSONFile(PackageFileTransform.Filename(), ds.Transform)
	return err
}
