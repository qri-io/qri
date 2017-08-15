package test

import (
	"github.com/qri-io/qri/repo"
	"testing"
)

type RepoTestFunc func(t *testing.T, r repo.Repo)

func RunRepoTests(t *testing.T, r repo.Repo) {
	tests := []RepoTestFunc{
		RunTestProfile,
		RunTestNamespace,
		RunTestQueryResults,
		RunTestResourceMeta,
		RunTestResourceQueries,
		RunTestPeers,
		RunTestDestroy,
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

func RunTestNamespace(t *testing.T, r repo.Repo) {
	g, err := r.Namespace()
	if err != nil {
		t.Errorf("Unexpected Namespace error: %s", err.Error())
		return
	}

	err = r.SaveNamespace(g)
	if err != nil {
		t.Errorf("Unexpected SaveNamespace error: %s", err.Error())
		return
	}
}

func RunTestQueryResults(t *testing.T, r repo.Repo) {
	g, err := r.QueryResults()
	if err != nil {
		t.Errorf("Unexpected QueryResults error: %s", err.Error())
		return
	}

	err = r.SaveQueryResults(g)
	if err != nil {
		t.Errorf("Unexpected SaveQueryResults error: %s", err.Error())
		return
	}
}

func RunTestResourceMeta(t *testing.T, r repo.Repo) {
	g, err := r.ResourceMeta()
	if err != nil {
		t.Errorf("Unexpected ResourceMeta error: %s", err.Error())
		return
	}

	err = r.SaveResourceMeta(g)
	if err != nil {
		t.Errorf("Unexpected SaveResourceMeta error: %s", err.Error())
		return
	}
}

func RunTestResourceQueries(t *testing.T, r repo.Repo) {
	g, err := r.ResourceQueries()
	if err != nil {
		t.Errorf("Unexpected ResourceQueries error: %s", err.Error())
		return
	}

	err = r.SaveResourceQueries(g)
	if err != nil {
		t.Errorf("Unexpected SaveResourceQueries error: %s", err.Error())
		return
	}
}

func RunTestPeers(t *testing.T, r repo.Repo) {
	p, err := r.Peers()
	if err != nil {
		t.Errorf("Unexpected Peers error: %s", err.Error())
		return
	}

	err = r.SavePeers(p)
	if err != nil {
		t.Errorf("Unexpected SavePeers error: %s", err.Error())
		return
	}
}

func RunTestDestroy(t *testing.T, r repo.Repo) {
	err := r.Destroy()
	if err != nil {
		t.Errorf("Unexpected Destroy error: %s", err.Error())
		return
	}
}
