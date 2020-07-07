package spec

import (
	"testing"

	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestNewTestRepo(t *testing.T) {
	rmf := func(t *testing.T) (repo.Repo, func()) {
		mr, err := repotest.NewEmptyTestRepo(event.NilBus)
		if err != nil {
			t.Fatal(err)
		}
		return mr, func() {}
	}

	RunRepoTests(t, rmf)
}

func TestNewMemRepoFromDir(t *testing.T) {
	repo, _, err := repotest.NewMemRepoFromDir("../testdata")
	if err != nil {
		t.Error(err.Error())
		return
	}

	c, err := repo.RefCount()
	if err != nil {
		t.Error(err.Error())
		return
	}

	// this should match count of valid testcases
	// in testdata
	expectRefCount := 6

	if c != expectRefCount {
		t.Errorf("expected %d datasets. got %d", expectRefCount, c)
	}
}

func TestNewTestRepoWithHistory(t *testing.T) {
	repo, log, err := repotest.NewTestRepoWithHistory()
	if err != nil {
		t.Fatal(err)
	}
	c, err := repo.RefCount()
	if err != nil {
		t.Error(err.Error())
		return
	}
	// there is only one ref that does not have a previous path:
	expectRefCount := 1
	if c != expectRefCount {
		t.Errorf("expected %d datasets, got %d", expectRefCount, c)
	}

	expectLogCount := 5
	if len(log) != expectLogCount {
		t.Errorf("expected %d datasets, got %d", expectLogCount, len(log))
	}

	for i, ref := range log {
		if ref.Name != "logtest" {
			t.Errorf("index %d, expected all datasets to have name 'logtest', got '%s'", i, ref.Name)
		}
	}
}
