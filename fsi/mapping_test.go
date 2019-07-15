package fsi

import (
	"fmt"
	"path/filepath"
	"testing"
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
			_, _, err := ReadDir(c.path)
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
			_, _, err := ReadDir(c.path)
			t.Log(err)
			if err == nil {
				t.Errorf("expected error. got: %s", err)
			}
		})
	}
}
