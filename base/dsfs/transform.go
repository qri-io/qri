package dsfs

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
)

// loadTransform assumes the provided path is correct
func loadTransform(ctx context.Context, store cafs.Filestore, path string) (q *dataset.Transform, err error) {
	data, err := fileBytes(store.Get(ctx, path))
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
func LoadTransformScript(ctx context.Context, store cafs.Filestore, dspath string) (qfs.File, error) {
	ds, err := LoadDataset(ctx, store, dspath)
	if err != nil {
		return nil, err
	}

	if ds.Transform == nil || ds.Transform.ScriptPath == "" {
		return nil, ErrNoTransform
	}

	return store.Get(ctx, ds.Transform.ScriptPath)
}
