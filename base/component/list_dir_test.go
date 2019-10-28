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
