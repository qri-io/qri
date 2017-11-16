package ipfs_filestore

import (
	"github.com/qri-io/cafs/test"
	"os"
	"path/filepath"
	"testing"
)

func TestFilestore(t *testing.T) {
	path := filepath.Join(os.TempDir(), "ipfs_cafs_test")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Errorf("error creating temp dir: %s", err.Error())
		return
	}
	defer os.RemoveAll(path)

	if err := InitRepo(path, ""); err != nil {
		t.Errorf("error intializing repo: %s", err.Error())
		return
	}

	f, err := NewFilestore(func(c *StoreCfg) {
		c.Online = false
		c.FsRepoPath = path
	})
	if err != nil {
		t.Errorf("error creating filestore: %s", err.Error())
		return
	}

	err = test.RunFilestoreTests(f)
	if err != nil {
		t.Errorf(err.Error())
	}
}
