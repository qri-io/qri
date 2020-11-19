package dsfs

import (
	"context"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
)

// LoadBody loads the data this dataset points to from the store
func LoadBody(ctx context.Context, fs qfs.Filesystem, ds *dataset.Dataset) (qfs.File, error) {
	return fs.Get(ctx, ds.BodyPath)
}
