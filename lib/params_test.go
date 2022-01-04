package lib

import (
	"path/filepath"
	"testing"
)

type testStruct struct {
	Name  string
	Path  string `qri:"fspath"`
	Ref   string
	Left  string `qri:"dsrefOrFspath"`
	Right string `qri:"dsrefOrFspath"`
}

func TestNormalizeInputParams(t *testing.T) {
	st := testStruct{
		Name:  "test_data",
		Path:  "testdata/dataset.yml",
		Ref:   "my_peer/my_dataset",
		Left:  "testdata/cities_2/body.csv",
		Right: "my_peer/another_ds",
	}
	normalizeInputParams(&st)

	if st.Name != "test_data" {
		t.Errorf("Name mismatch, expected: test_data, got: %s", st.Name)
	}
	if !filepath.IsAbs(st.Path) {
		t.Errorf("Path mismatch, expected abs path, got: %s", st.Path)
	}
	if st.Ref != "my_peer/my_dataset" {
		t.Errorf("Ref mismatch, expected: my_peer/my_dataset, got: %s", st.Ref)
	}
	if !filepath.IsAbs(st.Left) {
		t.Errorf("Left mismatch, expected abs path, got: %s", st.Left)
	}
	if st.Right != "my_peer/another_ds" {
		t.Errorf("Right mismatch, expected: my_peer/another_ds, got: %s", st.Right)
	}
}
