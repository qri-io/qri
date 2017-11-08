package test

import (
	"github.com/qri-io/qri/repo"
	"testing"
)

type RepoTestFunc func(t *testing.T, r repo.Repo)

func RunRepoTests(t *testing.T, r repo.Repo) {
	tests := []RepoTestFunc{
		RunTestProfile,
		// RunTestNamespace,
		// RunTestQueryResults,
		// RunTestResourceMeta,
		// RunTestResourceQueries,
		// RunTestPeers,
		// RunTestDestroy,
	}

	for _, test := range tests {
		test(t, r)
	}
}

func RunTestProfile(t *testing.T, r repo.Repo) {
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

func RunTestDatasetStore(t *testing.T, r repo.Repo) {
	// TODO
}

func RunTestPeers(t *testing.T, r repo.Repo) {
	// TODO
}

func RunTestAnalytics(t *testing.T, r repo.Repo) {
	// TODO
}

func RunTestCache(t *testing.T, r repo.Repo) {
	// TODO
}
