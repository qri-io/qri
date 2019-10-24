package dsfs

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs/cafs"
)

// loadMeta assumes the provided path is valid
func loadMeta(ctx context.Context, store cafs.Filestore, path string) (md *dataset.Meta, err error) {
	data, err := fileBytes(store.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading metadata file: %s", err.Error())
	}
	return dataset.UnmarshalMeta(data)
}
