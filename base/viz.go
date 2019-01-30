package base

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/fs"
	"github.com/qri-io/qri/repo"
)

// PrepareViz loads vizualization bytes from a local filepath
func prepareViz(r repo.Repo, ds *dataset.Dataset) (err error) {
	// remove any empty vizualizations
	if ds.Viz != nil && ds.Viz.IsEmpty() {
		ds.Viz = nil
		return nil
	}

	if ds.Viz != nil {
		// if ds.Viz.ScriptBytes != nil {
		// 	return
		// }
		if ds.Viz.ScriptBytes != nil {
			return
		}
		if ds.Viz.ScriptPath != "" {
			if strings.HasPrefix(ds.Viz.ScriptPath, "/ipfs") || strings.HasPrefix(ds.Viz.ScriptPath, "/map") || strings.HasPrefix(ds.Viz.ScriptPath, "/cafs") {
				var (
					f    fs.File
					data []byte
				)
				f, err = r.Store().Get(ds.Viz.ScriptPath)
				if err != nil {
					return
				}
				// ds.Viz.Script = f
				// TODO (b5): fix the file situation. This assignment is bad
				data, err = ioutil.ReadAll(f)
				if err != nil {
					return
				}
				ds.Viz.ScriptBytes = data
			} else {
				var (
					f    *os.File
					data []byte
				)
				f, err = os.Open(ds.Viz.ScriptPath)
				if err != nil {
					return
				}
				// ds.Viz.Script = f
				data, err = ioutil.ReadAll(f)
				if err != nil {
					ds.Viz.ScriptBytes = data
				}
			}
		}
	}
	return nil
}
