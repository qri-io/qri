package test

import (
	"testing"

	"github.com/qri-io/qri/repo"
)

func TestNewTestRepo(t *testing.T) {
	rmf := func(t *testing.T) (repo.Repo, func()) {
		mr, err := NewEmptyTestRepo(nil)
		if err != nil {
			t.Fatal(err)
		}
		return mr, func() {}
	}

	RunRepoTests(t, rmf)
}

func TestNewMemRepoFromDir(t *testing.T) {
	repo, _, err := NewMemRepoFromDir("testdata")
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
