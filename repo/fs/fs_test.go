package fsrepo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo/test"
)

func TestRepo(t *testing.T) {
	path := filepath.Join(os.TempDir(), "qri_repo_test")
	r, err := NewRepo(cafs.NewMapstore(), path, "test_repo_id")
	if err != nil {
		t.Errorf("error creating repo: %s", err.Error())
		return
	}

	test.RunRepoTests(t, r)

	if err := os.RemoveAll(path); err != nil {
		t.Errorf("error cleaning up after test: %s", err.Error())
	}
}
