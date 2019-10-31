package component

import (
	"testing"

	"github.com/qri-io/dataset"
)

func TestConvertDatasetToComponents(t *testing.T) {
	ds := dataset.Dataset{}
	ds.Meta = &dataset.Meta{
		Title: "test",
	}
	ds.Structure = &dataset.Structure{
		Format: "json",
	}

	comp := ConvertDatasetToComponents(&ds, nil)

	sub := comp.Base().GetSubcomponent("commit")
	if sub != nil {
		t.Errorf("expected nil commit")
	}

	sub = comp.Base().GetSubcomponent("body")
	if sub != nil {
		t.Errorf("expected nil body")
	}

	sub = comp.Base().GetSubcomponent("meta")
	if sub == nil {
		t.Fatalf("expected meta component")
	}
	meta, ok := sub.(*MetaComponent)
	if !ok {
		t.Fatalf("expected meta component type conversion to MetaComponent")
	}
	if meta.Value.Title != "test" {
		t.Errorf("expected meta.title \"%s\", got \"%s\"", "test", meta.Value.Title)
	}

	sub = comp.Base().GetSubcomponent("structure")
	if sub == nil {
		t.Fatalf("expected structure component")
	}
	structure, ok := sub.(*StructureComponent)
	if !ok {
		t.Fatalf("expected structure component type conversion to StructureComponent")
	}
	if structure.Value.Format != "json" {
		t.Errorf("expected structure.title \"%s\", got \"%s\"", "json", structure.Value.Format)
	}
}

func TestToDataset(t *testing.T) {
	dsComp := DatasetComponent{}
	dsComp.Base().SetSubcomponent(
		"commit",
		BaseComponent{
			Format:     "json",
			SourceFile: "testdata/commit.json",
		},
	)

	ds, err := ToDataset(&dsComp)
	if err != nil {
		t.Fatal(err)
	}
	if ds == nil {
		t.Fatal("ds is nil")
	}
	if ds.Commit == nil {
		t.Errorf("expected commit component")
	}
	if ds.Commit.Message != "test" {
		t.Errorf("expected commit.message \"%s\", got \"%s\"", "test", ds.Commit.Message)
	}
}
