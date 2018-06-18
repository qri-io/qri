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
		testRefSelector,
		// testRefstore,
		// DatasetActions,
	}

	for _, test := range tests {
		test(t, rmf)
	}
}

func testProfile(t *testing.T, rmf RepoMakerFunc) {
	r := rmf(t)
	p, err := r.Profile()
	if err != nil {
		t.Errorf("%s", string(p.ID))
		t.Errorf("Unexpected Profile error: %s", err.Error())
		return
	}

	err = r.SetProfile(p)
	if err != nil {
		t.Errorf("Unexpected SetProfile error: %s", err.Error())
		return
	}

}

func testRefSelector(t *testing.T, rmf RepoMakerFunc) {
	r := rmf(t)
	if rs, ok := r.(repo.RefSelector); ok {
		sel := []repo.DatasetRef{
			{Peername: "foo"},
		}

		err := rs.SetSelectedRefs(sel)
		if err != nil {
			t.Errorf("Error setting selection: %s", err)
		}

		got, err := rs.SelectedRefs()
		if len(sel) != len(got) {
			t.Errorf("Selected length mismatch. Expected: %d. Got: %d.", len(sel), len(got))
		}

		for i, a := range sel {
			if err := repo.CompareDatasetRef(a, got[i]); err != nil {
				t.Errorf("comparing selected reference %d: %s", i, err)
			}
		}
	}
}
