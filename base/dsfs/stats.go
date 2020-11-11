package dsfs

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
)

// DerefStats derferences a dataset's stats component if required
// no-op if ds.Stats is nil or isn't a reference
func DerefStats(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Stats != nil && ds.Stats.IsEmpty() && ds.Stats.Path != "" {
		sa, err := loadStats(ctx, store, ds.Stats.Path)
		if err != nil {
			log.Debug(err)
			return fmt.Errorf("loading stats component: %w", err)
		}
		ds.Stats = sa
	}
	return nil
}

// loadStats assumes the provided path is valid
func loadStats(ctx context.Context, fs qfs.Filesystem, path string) (sa *dataset.Stats, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("loading stats file: %w", err)
	}
	sa = &dataset.Stats{}
	err = sa.UnmarshalJSON(data)
	return sa, err
}
