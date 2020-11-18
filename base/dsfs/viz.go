package dsfs

import (
	"context"
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsviz"
	"github.com/qri-io/qfs"
)

// DerefViz dereferences a dataset's Viz element if required
// no-op if ds.Viz is nil or isn't a reference
func DerefViz(ctx context.Context, store qfs.Filesystem, ds *dataset.Dataset) error {
	if ds.Viz != nil && ds.Viz.IsEmpty() && ds.Viz.Path != "" {
		vz, err := loadViz(ctx, store, ds.Viz.Path)
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("loading dataset viz: %w", err)
		}
		vz.Path = ds.Viz.Path
		ds.Viz = vz
	}
	return nil
}

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

func vizFilesAddFunc(fs qfs.Filesystem, sw SaveSwitches) addWriteFileFunc {
	return func(ds *dataset.Dataset, wfs *writeFiles) error {
		if ds.Viz == nil {
			return nil
		}

		ds.Viz.DropTransientValues()

		vzfs := ds.Viz.ScriptFile()
		if vzfs != nil {
			wfs.vizScript = qfs.NewMemfileReader(PackageFileVizScript.Filename(), vzfs)
		}

		renderedF := ds.Viz.RenderedFile()
		if renderedF != nil {
			wfs.vizRendered = qfs.NewMemfileReader(PackageFileRenderedViz.Filename(), renderedF)
		} else if vzfs != nil && sw.ShouldRender {
			hook := renderVizWriteHook(fs, ds, wfs)
			wfs.vizRendered = qfs.NewWriteHookFile(emptyFile(PackageFileRenderedViz.Filename()), hook, append([]string{PackageFileVizScript.Filename()}, filePaths(wfs.files())...)...)
		}

		// we don't add the viz component itself, it's inlined in dataset.json
		return nil
	}
}

func renderVizWriteHook(fs qfs.Filesystem, ds *dataset.Dataset, wfs *writeFiles) qfs.WriteHook {
	return func(ctx context.Context, f qfs.File, added map[string]string) (io.Reader, error) {
		log.Debugf("running render hook")

		renderDs := &dataset.Dataset{}
		renderDs.Assign(ds)
		bf, err := fs.Get(ctx, added[wfs.body.FullPath()])
		if err != nil {
			return nil, err
		}
		sf, err := fs.Get(ctx, added[PackageFileVizScript.Filename()])
		if err != nil {
			log.Debugf("loading viz script file: %s", err)
			return nil, err
		}

		renderDs.SetBodyFile(bf)
		renderDs.Viz.SetScriptFile(sf)
		return dsviz.Render(renderDs)
	}
}
