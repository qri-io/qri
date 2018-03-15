package test

import (
	"testing"
)

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
	expectRefCount := 5

	if c != expectRefCount {
		t.Errorf("expected %d datasets. got %d", expectRefCount, c)
	}
}
