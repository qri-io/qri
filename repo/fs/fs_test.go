package fs_repo

import (
	"github.com/qri-io/qri/repo/test"
	"os"
	"path/filepath"
	"testing"
)

func TestRepo(t *testing.T) {
	r, err := NewRepo(filepath.Join(os.TempDir(), "qri_repo_test"))
	if err != nil {
		t.Errorf("error creating repo: %s", err.Error())
		return
	}

	test.RunRepoTests(t, r)
}
