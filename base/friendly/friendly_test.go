package friendly

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/deepdiff"
)

func TestFriendlyDiffDescriptions(t *testing.T) {
	// Change both the meta and structure
	deltas := []*deepdiff.Delta{
		&deepdiff.Delta{
			Type:        deepdiff.DTUpdate,
			Path:        "/meta/title",
			Value:       "def",
			SourcePath:  "/meta/title",
			SourceValue: "abc",
		},
		&deepdiff.Delta{
			Type:        deepdiff.DTUpdate,
			Path:        "/structure/formatConfig/headerRow",
			Value:       true,
			SourcePath:  "/structure/formatConfig/headerRow",
			SourceValue: false,
		},
	}
	stats := deepdiff.Stats{
		Left: 46,
	}

	shortTitle, longMessage := DiffDescriptions(deltas, &stats)
	expect := "updated meta and structure"
	if shortTitle != expect {
		t.Errorf("error comparing short title, expect: %s\ngot: %s", expect, shortTitle)
	}
	expect = `meta:
	updated title
structure:
	updated formatConfig.headerRow`
	if longMessage != expect {
		t.Errorf("error comparing long message, expect: %s\ngot: %s", expect, longMessage)
	}
}

func TestBuildComponentChanges(t *testing.T) {
	// Change the meta.title
	deltas := []*deepdiff.Delta{
		&deepdiff.Delta{
			Type:        deepdiff.DTUpdate,
			Path:        "/meta/title",
			Value:       "def",
			SourcePath:  "/meta/title",
			SourceValue: "abc",
		},
	}
	m := buildComponentChanges(deltas)
	keys := getKeys(m)
	expectList := []string{"meta"}
	if diff := cmp.Diff(expectList, keys); diff != "" {
		t.Fatalf("result mismatch (-want +got):%s\n", diff)
	}

	changes := m["meta"]
	expectList = []string{"updated title"}
	if diff := cmp.Diff(expectList, changes.Rows); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Change the structure
	deltas = []*deepdiff.Delta{
		&deepdiff.Delta{
			Type:        deepdiff.DTUpdate,
			Path:        "/structure/formatConfig/headerRow",
			Value:       true,
			SourcePath:  "/structure/formatConfig/headerRow",
			SourceValue: false,
		},
	}
	m = buildComponentChanges(deltas)
	keys = getKeys(m)
	expectList = []string{"structure"}
	if diff := cmp.Diff(expectList, keys); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	changes = m["structure"]
	expectList = []string{"updated formatConfig.headerRow"}
	if diff := cmp.Diff(expectList, changes.Rows); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Change both the meta and structure
	deltas = []*deepdiff.Delta{
		&deepdiff.Delta{
			Type:        deepdiff.DTUpdate,
			Path:        "/meta/title",
			Value:       "def",
			SourcePath:  "/meta/title",
			SourceValue: "abc",
		},
		&deepdiff.Delta{
			Type:        deepdiff.DTUpdate,
			Path:        "/structure/formatConfig/headerRow",
			Value:       true,
			SourcePath:  "/structure/formatConfig/headerRow",
			SourceValue: false,
		},
	}
	m = buildComponentChanges(deltas)
	keys = getKeys(m)
	expectList = []string{"meta", "structure"}
	if diff := cmp.Diff(expectList, keys); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}

func getKeys(m map[string]*ComponentChanges) []string {
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
