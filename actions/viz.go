package actions

import (
	"bytes"
	"io/ioutil"

	"github.com/qri-io/dataset"
)

// PrepareViz loads vizualization bytes from a local filepath
func (act Dataset) PrepareViz(ds *dataset.Dataset) (err error) {
	if ds.Viz != nil && ds.Viz.ScriptPath != "" {
		// create a reader of script bytes
		scriptdata, err := ioutil.ReadFile(ds.Viz.ScriptPath)
		if err != nil {
			return err
		}
		ds.Viz.Script = bytes.NewReader(scriptdata)
	}
	return nil
}
