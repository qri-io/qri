package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/qri/repo"
)

// RepoMakerFunc produces a new instance of a repository when called
type RepoMakerFunc func(t *testing.T) repo.Repo

// repoTestFunc is a function for testing a repo
type repoTestFunc func(t *testing.T, rm RepoMakerFunc)

func testdataPath(path string) string {
	return filepath.Join(os.Getenv("GOPATH"), "/src/github.com/qri-io/qri/repo/test/testdata", path)
}

// RunRepoTests tests that this repo conforms to
// expected behaviors
func RunRepoTests(t *testing.T, rmf RepoMakerFunc) {
	tests := []repoTestFunc{
		testProfile,
		testRefstore,
		DatasetActions,
	}

	for _, test := range tests {
		test(t, rmf)
	}
}

func testProfile(t *testing.T, rmf RepoMakerFunc) {
	r := rmf(t)
	p, err := r.Profile()
	if err != nil {
		t.Errorf("Unexpected Profile error: %s", err.Error())
		return
	}

	err = r.SaveProfile(p)
	if err != nil {
		t.Errorf("Unexpected SaveProfile error: %s", err.Error())
		return
	}

}
