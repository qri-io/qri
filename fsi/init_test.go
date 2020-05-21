package fsi

import (
	"testing"
)

func TestInitDataset(t *testing.T) {
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo, nil)

	_, err := fsi.InitDataset(InitParams{
		Name:   "test_ds",
		Dir:    paths.firstDir,
		Format: "csv",
	})
	if err != nil {
		t.Fatalf(err.Error())
	}
}
