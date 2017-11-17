package test

import (
	"fmt"
	"testing"

	"github.com/qri-io/qri/repo"
)

type RepoTestFunc func(r repo.Repo) error

func RunRepoTests(t *testing.T, r repo.Repo) {
	tests := []RepoTestFunc{
		RunTestProfile,
		RunTestNamespace,
		// RunTestQueryResults,
		// RunTestResourceMeta,
		// RunTestResourceQueries,
		// RunTestPeers,
		// RunTestDestroy,
	}

	for _, test := range tests {
		if err := test(r); err != nil {
			t.Errorf(err.Error())
		}
	}
}

func RunTestProfile(r repo.Repo) error {
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

func RunTestDatasetStore(r repo.Repo) error {
	// TODO
	return nil
}

func RunTestPeers(r repo.Repo) error {
	// TODO
	return nil
}

func RunTestAnalytics(r repo.Repo) error {
	// TODO
	return nil
}

func RunTestCache(r repo.Repo) error {
	// TODO
	return nil
}
