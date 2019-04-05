package lib

import (
	"testing"

	"github.com/qri-io/dataset"
)

func TestReadDatasetFiles(t *testing.T) {
	cases := []struct {
		description string
		paths       []string
		ds          *dataset.Dataset
	}{
		{".star file to transform script",
			[]string{
				"testdata/tf/transform.star",
			},
			&dataset.Dataset{
				Transform: &dataset.Transform{
					ScriptPath: "testdata/tf/transform.star",
				},
			},
		},

		{".html file to viz script",
			[]string{
				"testdata/viz/visualization.html",
			},
			&dataset.Dataset{
				Viz: &dataset.Viz{
					Format:     "html",
					ScriptPath: "testdata/viz/visualization.html",
				},
			},
		},

		{"structure.json, meta.json component files",
			[]string{
				"testdata/component_files/structure.json",
				"testdata/component_files/meta.json",
			},
			&dataset.Dataset{
				Meta: &dataset.Meta{
					Qri:   "md",
					Title: "build a dataset with component files",
				},
				Structure: &dataset.Structure{
					Qri:    "st",
					Format: "json",
					Schema: map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "array",
							"items": []interface{}{
								map[string]interface{}{"type": "string", "name": "field_1"},
							},
						},
					},
				},
			},
		},

		{"structure.yaml, meta.yaml component files",
			[]string{
				"testdata/component_files/structure.yaml",
				"testdata/component_files/meta.yaml",
			},
			&dataset.Dataset{
				Meta: &dataset.Meta{
					Qri:   "md",
					Title: "build a dataset with component files",
				},
				Structure: &dataset.Structure{
					Qri:    "st",
					Format: "json",
					Schema: map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "array",
							"items": []interface{}{
								map[string]interface{}{"type": "string", "name": "field_1"},
							},
						},
					},
				},
			},
		},

		{"structure.json, meta.yaml, commit.yaml component files",
			[]string{
				"testdata/component_files/commit.yaml",
				"testdata/component_files/structure.json",
				"testdata/component_files/meta.yaml",
			},
			&dataset.Dataset{
				Commit: &dataset.Commit{
					Qri:   "cm",
					Title: "this is a commit",
				},
				Meta: &dataset.Meta{
					Qri:   "md",
					Title: "build a dataset with component files",
				},
				Structure: &dataset.Structure{
					Qri:    "st",
					Format: "json",
					Schema: map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "array",
							"items": []interface{}{
								map[string]interface{}{"type": "string", "name": "field_1"},
							},
						},
					},
				},
			},
		},
	}

	for i, c := range cases {
		got, err := ReadDatasetFiles(c.paths...)
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
