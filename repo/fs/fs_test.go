package fsrepo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/test"
)

func TestRepo(t *testing.T) {
	path := filepath.Join(os.TempDir(), "qri_repo_test")
	t.Log(path)

	rmf := func(t *testing.T) repo.Repo {
		if err := os.RemoveAll(path); err != nil {
			t.Errorf("error removing files: %s", err.Error())
		}

		pro, err := profile.NewProfile(config.DefaultProfile())
		if err != nil {
			t.Error(err.Error())
		}

		r, err := NewRepo(cafs.NewMapstore(), pro, nil, path)
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
