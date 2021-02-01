// Package spec contains a set of tests to ensure a repo implementation conforms
// to expected behaviors, calling RunRepoTests on a given repo implementation should
// pass all checks in order to properly work with Qri.
// test also has a TestRepo, which uses an in-memory implementation of Repo
// suited for tests that require a repo
package spec

import (
	"context"
	"testing"

	"github.com/qri-io/qri/repo"
)

// RepoMakerFunc produces a new instance of a repository when called
// the returned cleanup function will be called at the end of each test,
// and can be used to do things like remove temp files
type RepoMakerFunc func(t *testing.T) (r repo.Repo, cleanup func())

// repoTestFunc is a function for testing a repo
type repoTestFunc func(t *testing.T, rm RepoMakerFunc)

// RunRepoTests tests that this repo conforms to expected behaviors
func RunRepoTests(t *testing.T, rmf RepoMakerFunc) {
	tests := map[string]repoTestFunc{
		"testProfile":             testProfile,
		"testRefstoreInvalidRefs": testRefstoreInvalidRefs,
		"testRefstoreRefs":        testRefstoreRefs,
		"testRefstore":            testRefstoreMain,
		"testProfileStore":        testProfileStore,
	}

	for key, test := range tests {
		t.Run(key, func(t *testing.T) {
			test(t, rmf)
		})
	}
}

func testProfile(t *testing.T, rmf RepoMakerFunc) {
	r, cleanup := rmf(t)
	defer cleanup()
	ctx := context.Background()

	p, err := r.Profile(ctx)
	if err != nil {
		t.Errorf("%s", string(p.ID))
		t.Errorf("Unexpected Profile error: %s", err.Error())
		return
	}

	err = r.Profiles().SetOwner(p)
	if err != nil {
		t.Errorf("Unexpected SetProfile error: %s", err.Error())
		return
	}

}
