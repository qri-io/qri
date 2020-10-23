package dsfs

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
)

// loadMeta assumes the provided path is valid
func loadMeta(ctx context.Context, fs qfs.Filesystem, path string) (md *dataset.Meta, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading metadata file: %s", err.Error())
	}
	return dataset.UnmarshalMeta(data)
}
