package component

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestListDirectoryComponents(t *testing.T) {
	components, err := ListDirectoryComponents("../../fsi/testdata/valid_mappings/some_json_components/")
	if err != nil {
		t.Fatalf(err.Error())
	}

	names := getComponentNames(components)
	expect := []string{"body", "meta", "readme"}
	if diff := cmp.Diff(expect, names); diff != "" {
		t.Fatalf("component names (-want +got):\n%s", diff)
	}

	err = ExpandListedComponents(components, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}

	bodyComponent := components.Base().GetSubcomponent("body").(*BodyComponent)
	bodyComponent.LoadAndFill(nil)
	data, err := json.Marshal(bodyComponent.Value)
	if err != nil {
		t.Fatalf(err.Error())
	}
	expectStr := "[[1,2,3],[4,5,6]]"
	if diff := cmp.Diff(expectStr, string(data)); diff != "" {
		t.Errorf("body component (-want +got):\n%s", diff)
	}

	metaComponent := components.Base().GetSubcomponent("meta").(*MetaComponent)
	metaComponent.LoadAndFill(nil)
	data, err = json.Marshal(metaComponent.Value)
	if err != nil {
		t.Fatalf(err.Error())
	}
	expectStr = "{\"qri\":\"md:0\",\"title\":\"title\"}"
	if diff := cmp.Diff(expectStr, string(data)); diff != "" {
		t.Errorf("meta component (-want +got):\n%s", diff)
	}

	readmeComponent := components.Base().GetSubcomponent("readme").(*ReadmeComponent)
	readmeComponent.LoadAndFill(nil)
	expectStr = "# Readme\n\nDescribes this dataset.\n"
	if diff := cmp.Diff(expectStr, string(readmeComponent.Value.ScriptBytes)); diff != "" {
		t.Errorf("readme component (-want +got):\n%s", diff)
	}
}

func TestIsKnownFilename(t *testing.T) {
	known := GetKnownFilenames()

	goodCases := []struct {
		description string
		filename    string
	}{
		{"body csv file", "body.csv"},
		{"meta json file", "meta.json"},
		{"readme file", "readme.md"},
		{"transform file", "transform.star"},
		{"dataset json", "dataset.json"},
		{"dataset yaml", "dataset.yaml"},
	}

	badCases := []struct {
		description string
		filename    string
	}{
		{"body bad extension", "body.bin"},
		{"meta bad extension", "meta.jpg"},
		{"unknown filename", "my_content.csv"},
		{"vi temporary", ".body.csv.swp"},
		{"emacs temporary", "#body.csv"},
	}

	for i, c := range goodCases {
		if !IsKnownFilename(c.filename, known) {
			t.Errorf("error for good case %d: %s", i, c.description)
		}
	}
	for i, c := range badCases {
		if IsKnownFilename(c.filename, known) {
			t.Errorf("error for bad case %d: %s", i, c.description)
		}
	}
}

func getComponentNames(comp Component) []string {
	names := make([]string, 0)
	for _, name := range AllSubcomponentNames() {
		sub := comp.Base().GetSubcomponent(name)
		if sub != nil {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func TestIsKnownFilenameAllowNil(t *testing.T) {
	goodFilename := "structure.json"
	badFilename := "structure.format"
	if !IsKnownFilename(goodFilename, nil) {
		t.Errorf("expected goodFilename to be a known filename")
	}
	if IsKnownFilename(badFilename, nil) {
		t.Errorf("expected badFilename to not be a known filename")
	}
}
