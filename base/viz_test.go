package base

import (
	"io/ioutil"
	"testing"

	"github.com/qri-io/cafs"

	"github.com/qri-io/dataset"
)

func TestPrepareViz(t *testing.T) {
	r := newTestRepo(t)
	tmpl := []byte(`<html><head><title>hallo</title></head></html>`)

	f, err := ioutil.TempFile("", "viz")
	if err != nil {
		t.Fatal(err.Error())
	}
	f.Write(tmpl)

	ds := &dataset.Dataset{
		Viz: &dataset.Viz{
			ScriptPath: f.Name(),
		},
	}

	if err := prepareViz(r, ds); err != nil {
		t.Error(err.Error())
	}

	key, err := r.Store().Put(cafs.NewMemfileBytes("tmpl.html", tmpl), true)
	if err != nil {
		t.Fatal(key)
	}

	ds = &dataset.Dataset{
		Viz: &dataset.Viz{
			ScriptPath: key.String(),
		},
	}

	if err := prepareViz(r, ds); err != nil {
		t.Error(err.Error())
	}
}
