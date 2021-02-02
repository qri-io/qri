package fsi

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitDataset(t *testing.T) {
	ctx := context.Background()
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo, nil)

	_, err := fsi.InitDataset(ctx, InitParams{
		Name:      "test_ds",
		TargetDir: paths.firstDir,
		Format:    "csv",
	})
	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestSplitDirByExist(t *testing.T) {
	tmps := NewTmpPaths()
	defer tmps.Close()

	// Create one directory
	if err := os.Mkdir(filepath.Join(tmps.firstDir, "create"), 0755); err != nil {
		t.Fatal(err)
	}
	// Split a path with two components after the existing directory
	subjectDir := filepath.Join(tmps.firstDir, "create/this/path")
	foundDir, missingDir := SplitDirByExist(subjectDir)

	if !strings.HasSuffix(foundDir, "/create") {
		t.Errorf("expected foundDir to have suffix \"/create\", got: %q", foundDir)
	}
	expectDir := "this/path"
	if missingDir != expectDir {
		t.Errorf("expected: %q, got: %q", expectDir, missingDir)
	}

	// Create another directory
	if err := os.Mkdir(filepath.Join(tmps.firstDir, "create/this"), 0755); err != nil {
		t.Fatal(err)
	}
	// Split again
	subjectDir = filepath.Join(tmps.firstDir, "create/this/path")
	foundDir, missingDir = SplitDirByExist(subjectDir)

	if !strings.HasSuffix(foundDir, "/create/this") {
		t.Errorf("expected foundDir to have suffix \"/create/this\", got: %q", foundDir)
	}
	expectDir = "path"
	if missingDir != expectDir {
		t.Errorf("expected: %q, got: %q", expectDir, missingDir)
	}

	// Create last directory
	if err := os.Mkdir(filepath.Join(tmps.firstDir, "create/this/path"), 0755); err != nil {
		t.Fatal(err)
	}
	// Split again
	subjectDir = filepath.Join(tmps.firstDir, "create/this/path")
	foundDir, missingDir = SplitDirByExist(subjectDir)

	if !strings.HasSuffix(foundDir, "/create/this/path") {
		t.Errorf("expected foundDir to have suffix \"/create/this/path\", got: %q", foundDir)
	}
	expectDir = ""
	if missingDir != expectDir {
		t.Errorf("expected: %q, got: %q", expectDir, missingDir)
	}
}
