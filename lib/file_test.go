package lib

import (
	"testing"

	"github.com/qri-io/dataset"
)

func TestReadDatasetFiles(t *testing.T) {
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
					Format:     "html",
					ScriptPath: "testdata/viz/visualization.html",
				},
			},
		},
	}

	for i, c := range cases {
		got, err := ReadDatasetFiles(c.path)
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
