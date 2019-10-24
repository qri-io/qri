package dsfs

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs/cafs"
)

// loadViz assumes the provided path is valid
func loadViz(ctx context.Context, store cafs.Filestore, path string) (st *dataset.Viz, err error) {
	data, err := fileBytes(store.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading viz file: %s", err.Error())
	}
	return dataset.UnmarshalViz(data)
}

// ErrNoViz is the error for asking a dataset without a viz component for viz info
var ErrNoViz = fmt.Errorf("this dataset has no viz component")
