package lib

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/dataset"
)

func TestAbsPath(t *testing.T) {
	tmp, err := filepath.Abs(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	pathAbs, err := filepath.Abs("relative/path/data.yaml")
	if err != nil {
		t.Fatal(err)
	}

	httpAbs, err := filepath.Abs("http_got/relative/dataset.yaml")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		in, out, err string
	}{
		{"", "", ""},
		{"http://example.com/zipfile.zip", "http://example.com/zipfile.zip", ""},
		{"https://example.com/zipfile.zip", "https://example.com/zipfile.zip", ""},
		{"relative/path/data.yaml", pathAbs, ""},
		{"http_got/relative/dataset.yaml", httpAbs, ""},
		{"/ipfs", "/ipfs", ""},
		{tmp, tmp, ""},
	}

	for i, c := range cases {
		got := c.in
		err := AbsPath(&got)
		if !(err == nil && c.err == "" || (err != nil && c.err == err.Error())) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
		}
		if got != c.out {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.out, got)
		}
	}
}

func TestReadDatasetFile(t *testing.T) {
	cases := []struct {
		description string
		path        string
		ds          *dataset.Dataset
	}{
		{".star file to transform script",
			"testdata/tf/transform.star",
			&dataset.Dataset{
				Transform: &dataset.Transform{
					ScriptPath: "testdata/tf/transform.star",
				},
			},
		},

		{".html file to viz script",
			"testdata/viz/visualization.html",
			&dataset.Dataset{
				Viz: &dataset.Viz{
					ScriptPath: "testdata/viz/visualization.html",
				},
			},
		},
	}

	for i, c := range cases {
		got, err := ReadDatasetFiles([]string{c.path})
		if err != nil {
			t.Errorf("case %d %s unexpected error: %s", i, c.description, err.Error())
			continue
		}
		if err := dataset.CompareDatasets(c.ds, got); err != nil {
			t.Errorf("case %d %s dataset mismatch: %s", i, c.description, err.Error())
			continue
		}
	}
}
