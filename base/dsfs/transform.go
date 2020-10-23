package dsfs

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
)

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
