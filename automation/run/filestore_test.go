package run_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/spec"
)

func TestFileStore(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	store, err := run.NewFileStore(tmpdir)
	if err != nil {
		t.Fatal(err)
	}

	spec.AssertRunStore(t, store)
}
