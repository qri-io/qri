package base

import (
	"os"
	"strings"

	datastore "github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo"
)

// PrepareViz loads vizualization bytes from a local filepath
func prepareViz(r repo.Repo, ds *dataset.Dataset) (err error) {
	// remove any empty vizualizations
	if ds.Viz != nil && ds.Viz.IsEmpty() {
		ds.Viz = nil
		return nil
	}

	if ds.Viz != nil && ds.Viz.ScriptPath != "" {
		if strings.HasPrefix(ds.Viz.ScriptPath, "/ipfs") || strings.HasPrefix(ds.Viz.ScriptPath, "/map") || strings.HasPrefix(ds.Viz.ScriptPath, "/cafs") {
			var f cafs.File
			f, err = r.Store().Get(datastore.NewKey(ds.Viz.ScriptPath))
			if err != nil {
				return
			}
			ds.Viz.Script = f
		} else {
			var f *os.File
			f, err = os.Open(ds.Viz.ScriptPath)
			if err != nil {
				return
			}
			ds.Viz.Script = f
		}
	}
	return nil
}
