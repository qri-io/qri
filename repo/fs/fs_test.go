package fs_repo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/qri/repo/test"
)

func TestRepo(t *testing.T) {
	r, err := NewRepo(memfs.NewMapstore(), filepath.Join(os.TempDir(), "qri_repo_test"), "test_repo_id")
	if err != nil {
		t.Errorf("error creating repo: %s", err.Error())
		return
	}

	test.RunRepoTests(t, r)
}
