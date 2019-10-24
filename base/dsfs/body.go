package dsfs

import (
	"context"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
)

// LoadBody loads the data this dataset points to from the store
func LoadBody(ctx context.Context, store cafs.Filestore, ds *dataset.Dataset) (qfs.File, error) {
	return store.Get(ctx, ds.BodyPath)
}
