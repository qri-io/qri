package dsfs

import (
	"context"
	"fmt"

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
