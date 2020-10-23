package dsfs

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
)

// loadReadme assumes the provided path is valid
func loadReadme(ctx context.Context, fs qfs.Filesystem, path string) (st *dataset.Readme, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading readme file: %s", err.Error())
	}
	return dataset.UnmarshalReadme(data)
}

// ErrNoReadme is the error for asking a dataset without a readme component for readme info
var ErrNoReadme = fmt.Errorf("this dataset has no readme component")

// LoadReadmeScript loads script data from a dataset path if the given dataset has a readme script is specified
// the returned qfs.File will be the value of dataset.Readme.ScriptPath
func LoadReadmeScript(ctx context.Context, fs qfs.Filesystem, dspath string) (qfs.File, error) {
	ds, err := LoadDataset(ctx, fs, dspath)
	if err != nil {
		return nil, err
	}

	if ds.Readme == nil || ds.Readme.ScriptPath == "" {
		return nil, ErrNoReadme
	}

	return fs.Get(ctx, ds.Readme.ScriptPath)
}
