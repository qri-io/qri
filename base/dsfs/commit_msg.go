package dsfs

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs/cafs"
)

// loadCommit assumes the provided path is valid
func loadCommit(ctx context.Context, store cafs.Filestore, path string) (st *dataset.Commit, err error) {
	data, err := fileBytes(store.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading commit file: %s", err.Error())
	}
	return dataset.UnmarshalCommit(data)
}
