package fsi

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReadDir(t *testing.T) {
	good := []struct {
		path string
	}{
		{"testdata/valid_mappings/all_json_components"},
		{"testdata/valid_mappings/all_in_dataset"},
	}

	for _, c := range good {
		t.Run(fmt.Sprintf("good: %s", filepath.Base(c.path)), func(t *testing.T) {
			_, _, _, err := ReadDir(c.path)
			if err != nil {
				t.Errorf("expected no error. got: %s", err)
			}
		})
	}

	bad := []struct {
		path string
	}{
		{"testdata/invalid_mappings/two_metas"},
		{"testdata/invalid_mappings/double_format"},
		{"testdata/invalid_mappings/bad_yaml"},
		{"testdata/invalid_mappings/empty"},
	}

	for _, c := range bad {
		t.Run(fmt.Sprintf("bad: %s", filepath.Base(c.path)), func(t *testing.T) {
			_, _, _, err := ReadDir(c.path)
			t.Log(err)
			if err == nil {
				t.Errorf("expected error. got: %s", err)
			}
		})
	}
}

func TestDeleteDatasetFiles(t *testing.T) {
	paths := NewTmpPaths()
	defer paths.Close()

	// first dir
	if err := ioutil.WriteFile(filepath.Join(paths.firstDir, "body.csv"), []byte(`first,second,third`), 0666); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(paths.firstDir, "example.ipnb"), []byte(`a file that isn't tracked by qri`), 0666); err != nil {
		t.Fatal(err)
	}

	bodyStat := tFileStat(t, paths.firstDir, "body.csv")

	mapping, err := DeleteDatasetFiles(paths.firstDir)
	if err != nil {
		t.Errorf("unexpected error deleting files: %s", err)
	}

	expect := map[string]FileStat{
		"body": bodyStat,
	}
	if diff := cmp.Diff(expect, mapping); diff != "" {
		t.Errorf("deleted file map mismatch (-want +got):\n%s", diff)
	}

	if _, err := os.Stat(paths.firstDir); err != nil {
		t.Errorf("expected first dir to still exist after deleting. got err: %s", err)
	}

	if err := ioutil.WriteFile(filepath.Join(paths.secondDir, "body.csv"), []byte(`first,second,third`), 0666); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(paths.secondDir, "structure.json"), []byte(`{"schema":{"type": "array"}}`), 0666); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(paths.secondDir, "meta.json"), []byte(`invalid, but existing metadata file`), 0666); err != nil {
		t.Fatal(err)
	}

	bodyStat = tFileStat(t, paths.secondDir, "body.csv")
	structureStat := tFileStat(t, paths.secondDir, "structure.json")
	metaStat := tFileStat(t, paths.secondDir, "meta.json")

	if mapping, err = DeleteDatasetFiles(paths.secondDir); err != nil {
		t.Errorf("unexpected error deleting files: %s", err)
	}

	expect = map[string]FileStat{
		"body":      bodyStat,
		"structure": structureStat,
		"meta":      metaStat,
	}
	if diff := cmp.Diff(expect, mapping); diff != "" {
		t.Errorf("deleted file map mismatch (-want +got):\n%s", diff)
	}

	if _, err := os.Stat(paths.secondDir); err == nil {
		t.Errorf("expected second dir to not exist after deleting")
	}
}
