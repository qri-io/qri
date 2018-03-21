package fsrepo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/test"
)

func TestRepo(t *testing.T) {
	path := filepath.Join(os.TempDir(), "qri_repo_test")

	rmf := func(t *testing.T) repo.Repo {
		if err := os.RemoveAll(path); err != nil {
			t.Errorf("error removing files: %s", err.Error())
		}

		r, err := NewRepo(cafs.NewMapstore(), path, "test_repo_id")
		if err != nil {
			t.Errorf("error creating repo: %s", err.Error())
		}
		return r
	}

	test.RunRepoTests(t, rmf)

	if err := os.RemoveAll(path); err != nil {
		t.Errorf("error cleaning up after test: %s", err.Error())
	}
}
