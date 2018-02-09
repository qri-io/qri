package test

import (
	"fmt"
	"testing"

	"github.com/qri-io/qri/repo"
)

// RepoTestFunc is a function for testing a repo
type RepoTestFunc func(r repo.Repo) error

// RunRepoTests tests that this repo conforms to
// expected behaviors
func RunRepoTests(t *testing.T, r repo.Repo) {
	tests := []RepoTestFunc{
		runTestProfile,
		runTestRefstore,
		// runTestQueryResults,
		// runTestResourceMeta,
		// runTestResourceQueries,
		// runTestPeers,
		// runTestDestroy,
	}

	for _, test := range tests {
		if err := test(r); err != nil {
			t.Errorf(err.Error())
		}
	}
}

func runTestProfile(r repo.Repo) error {
	p, err := r.Profile()
	if err != nil {
		return fmt.Errorf("Unexpected Profile error: %s", err.Error())
	}

	err = r.SaveProfile(p)
	if err != nil {
		return fmt.Errorf("Unexpected SaveProfile error: %s", err.Error())
	}
	return nil
}

func runTestDatasetStore(r repo.Repo) error {
	// TODO
	return nil
}

func runTestPeers(r repo.Repo) error {
	// TODO
	return nil
}

func runTestAnalytics(r repo.Repo) error {
	// TODO
	return nil
}

func runTestCache(r repo.Repo) error {
	// TODO
	return nil
}
