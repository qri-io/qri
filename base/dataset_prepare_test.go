package base

import (
	"encoding/json"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/fs"
)

func TestPrepareDatasetSave(t *testing.T) {
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	prev, mutable, body, prevPath, err := PrepareDatasetSave(r, ref.Peername, ref.Name)
	if err != nil {
		t.Errorf("case cities dataset error: %s ", err.Error())
	}
	if prev.IsEmpty() {
		t.Errorf("case cites dataset: previous should not be empty")
	}
	if mutable.IsEmpty() {
		t.Errorf("case cities dataset: mutable should not be empty")
	}
	if mutable.Transform != nil {
		t.Errorf("case cities dataset: mutable.Transform should be nil")
	}
	if mutable.Commit != nil {
		t.Errorf("case cities dataset: mutable.Commit should be nil")
	}
	if body == nil {
		t.Errorf("case cities dataset: previous body should not be nil")
	}
	if prevPath == "" {
		t.Errorf("case cities dataset: previous path should not be empty")
	}

	prev, mutable, body, prevPath, err = PrepareDatasetSave(r, "me", "non-existent")
	if err != nil {
		t.Errorf("case non-existant previous dataset error: %s ", err.Error())
	}
	if !prev.IsEmpty() {
		t.Errorf("case non-existant previous dataset: previous should be empty, got non-empty dataset")
	}
	if !mutable.IsEmpty() {
		t.Errorf("case non-existant previous dataset: mutable should be empty, got non-empty dataset")
	}
	if body != nil {
		t.Errorf("case non-existant previous dataset: previous body should be nil, got non-nil body")
	}
	if prevPath != "" {
		t.Errorf("case non-existant previous dataset: previous path should be empty, got non-empty path")
	}
}

func TestInferValuesDatasetName(t *testing.T) {
	r := newTestRepo(t)
	pro, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}

	name := ""
	body := fs.NewMemfileBytes("gabba gabba hey.csv", []byte("a,b,c,c,s,v"))
	ds := &dataset.Dataset{}
	if _, err = InferValues(pro, &name, ds, body); err != nil {
		t.Error(err)
	}
	expectName := "gabba_gabba_heycsv"
	if expectName != name {
		t.Errorf("inferred name mismatch. expected: '%s', got: '%s'", expectName, name)
	}
}

func TestInferValuesStructure(t *testing.T) {
	r := newTestRepo(t)
	pro, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}

	name := "animals"
	body := fs.NewMemfileBytes("animals.csv",
		[]byte("Animal,Sound,Weight\ncat,meow,1.4\ndog,bark,3.7\n"))
	ds := &dataset.Dataset{}

	if _, err = InferValues(pro, &name, ds, body); err != nil {
		t.Error(err)
	}

	if ds.Structure.Format != "csv" {
		t.Errorf("expected format CSV, got %s", ds.Structure.Format)
	}
	if ds.Structure.FormatConfig["headerRow"] != true {
		t.Errorf("expected format config to set headerRow set to true")
	}

	actual := datasetSchemaToJSON(ds)
	expect := `{"items":{"items":[{"title":"animal","type":"string"},{"title":"sound","type":"string"},{"title":"weight","type":"number"}],"type":"array"},"type":"array"}`

	if expect != actual {
		t.Errorf("mismatched schema, expected \"%s\", got \"%s\"", expect, actual)
	}
}

func TestInferValuesSchema(t *testing.T) {
	r := newTestRepo(t)
	pro, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}

	name := "animals"
	body := fs.NewMemfileBytes("animals.csv",
		[]byte("Animal,Sound,Weight\ncat,meow,1.4\ndog,bark,3.7\n"))
	ds := &dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "csv",
		},
	}
	if _, err = InferValues(pro, &name, ds, body); err != nil {
		t.Error(err)
	}

	if ds.Structure.Format != "csv" {
		t.Errorf("expected format CSV, got %s", ds.Structure.Format)
	}
	if ds.Structure.FormatConfig["headerRow"] != true {
		t.Errorf("expected format config to set headerRow set to true")
	}

	actual := datasetSchemaToJSON(ds)
	expect := `{"items":{"items":[{"title":"animal","type":"string"},{"title":"sound","type":"string"},{"title":"weight","type":"number"}],"type":"array"},"type":"array"}`

	if expect != actual {
		t.Errorf("mismatched schema, expected \"%s\", got \"%s\"", expect, actual)
	}
}

func TestInferValuesDontOverwriteSchema(t *testing.T) {
	r := newTestRepo(t)
	pro, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}

	name := "animals"
	body := fs.NewMemfileBytes("animals.csv",
		[]byte("Animal,Sound,Weight\ncat,meow,1.4\ndog,bark,3.7\n"))
	ds := &dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "csv",
			Schema: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "array",
					"items": []interface{}{
						map[string]interface{}{"title": "animal", "type": "number"},
						map[string]interface{}{"title": "noise", "type": "number"},
						map[string]interface{}{"title": "height", "type": "number"},
					},
				},
			},
		},
	}
	if _, err = InferValues(pro, &name, ds, body); err != nil {
		t.Error(err)
	}

	if ds.Structure.Format != "csv" {
		t.Errorf("expected format CSV, got %s", ds.Structure.Format)
	}
	if ds.Structure.FormatConfig != nil {
		t.Errorf("expected format config to be nil")
	}

	actual := datasetSchemaToJSON(ds)
	expect := `{"items":{"items":[{"title":"animal","type":"number"},{"title":"noise","type":"number"},{"title":"height","type":"number"}],"type":"array"},"type":"array"}`

	if expect != actual {
		t.Errorf("mismatched schema, expected \"%s\", got \"%s\"", expect, actual)
	}
}

func TestValidateDataset(t *testing.T) {
	if err := ValidateDataset("this name has spaces", nil); err == nil {
		t.Errorf("expected invalid name to fail")
	}
}

func datasetSchemaToJSON(ds *dataset.Dataset) string {
	js, err := json.Marshal(ds.Structure.Schema)
	if err != nil {
		return err.Error()
	}
	return string(js)
}
