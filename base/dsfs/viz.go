package dsfs

import (
	"context"
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsviz"
	"github.com/qri-io/qfs"
)

// loadViz assumes the provided path is valid
func loadViz(ctx context.Context, fs qfs.Filesystem, path string) (st *dataset.Viz, err error) {
	data, err := fileBytes(fs.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading viz file: %s", err.Error())
	}
	return dataset.UnmarshalViz(data)
}

// ErrNoViz is the error for asking a dataset without a viz component for viz info
var ErrNoViz = fmt.Errorf("this dataset has no viz component")

func renderVizWriteHook(fs qfs.Filesystem, ds *dataset.Dataset, bodyFilename string) qfs.WriteHook {
	return func(ctx context.Context, f qfs.File, added map[string]string) (io.Reader, error) {
		log.Debugf("running render hook")

		renderDs := &dataset.Dataset{}
		renderDs.Assign(ds)
		bf, err := fs.Get(ctx, added[bodyFilename])
		if err != nil {
			return nil, err
		}
		sf, err := fs.Get(ctx, added["viz_script"])
		if err != nil {
			return nil, err
		}

		renderDs.SetBodyFile(bf)
		renderDs.Viz.SetScriptFile(sf)
		return dsviz.Render(renderDs)
	}
}
