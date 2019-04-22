package cron

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestFsJobStore(t *testing.T) {
	tmp, err := ioutil.TempDir(os.TempDir(), "TestFsJobStore")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	newStore := func() JobStore {
		return NewFileJobStore(filepath.Join(tmp, "jobs.cbor"))
	}
	RunJobStoreTests(t, newStore)
}
