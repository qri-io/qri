package base

import (
	"io/ioutil"
	"testing"

	"github.com/qri-io/dataset"
)

func TestPrepareViz(t *testing.T) {
	f, err := ioutil.TempFile("", "viz")
	if err != nil {
		t.Fatal(err.Error())
	}
	f.Write([]byte(`<html><head><title>hallo</title></head></html>`))

	ds := &dataset.Dataset{
		Viz: &dataset.Viz{
			ScriptPath: f.Name(),
		},
	}

	if err := PrepareViz(ds); err != nil {
		t.Error(err.Error())
	}
}
