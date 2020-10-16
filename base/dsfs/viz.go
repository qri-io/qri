package dsfs

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsviz"
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

func renderVizMerkleHook(dslk *sync.Mutex, ds *dataset.Dataset, bodyFilename string) *MerkelizeHook {
	cb := func(ctx context.Context, store cafs.Filestore, merkelizedPaths map[string]string) (io.Reader, error) {
		log.Debugf("running render merkelize hook")
		dslk.Lock()
		defer dslk.Unlock()

		renderDs := &dataset.Dataset{}
		renderDs.Assign(ds)
		bf, err := store.Get(ctx, merkelizedPaths[bodyFilename])
		if err != nil {
			return nil, err
		}
		sf, err := store.Get(ctx, merkelizedPaths["viz_script"])
		if err != nil {
			return nil, err
		}

		log.Debugf("%#v", renderDs)
		renderDs.SetBodyFile(bf)
		renderDs.Viz.SetScriptFile(sf)
		return dsviz.Render(ds)
	}

	return NewMerkelizeHook(
		PackageFileRenderedViz.String(),
		cb,
		bodyFilename,
		PackageFileStructure.String(),
		PackageFileMeta.String(),
		"viz_script",
	)
}
