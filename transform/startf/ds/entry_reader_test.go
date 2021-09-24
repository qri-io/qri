package ds

import (
	"fmt"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"go.starlark.net/starlark"
)

// assert *EntryReader conforms to dsio.EntryReader interface
var _ dsio.EntryReader = (*EntryReader)(nil)

func TestEntryReaderSimpleList(t *testing.T) {
	var elems *starlark.List
	elems = starlark.NewList([]starlark.Value{})
	elems.Append(starlark.MakeInt(1))
	elems.Append(starlark.MakeInt(2))
	elems.Append(starlark.MakeInt(3))
	st := &dataset.Structure{
		Schema: dataset.BaseSchemaArray,
	}
	r := NewEntryReader(st, elems)

	expect := []struct {
		index int
		key   string
		value string
	}{
		{0, "", "1"},
		{1, "", "2"},
		{2, "", "3"},
	}

	for i, e := range expect {
		ent, err := r.ReadEntry()
		if err != nil {
			log.Fatal(err)
		}

		if e.index != ent.Index {
			t.Errorf("case %d: index did not match, expect: %d, actual: %d", i, e.index, ent.Index)
		}
		if e.key != ent.Key {
			t.Errorf("case %d: key did not match, expect: %s, actual: %s", i, e.key, ent.Key)
		}
		val := fmt.Sprintf("%v", ent.Value)
		if e.value != val {
			t.Errorf("case %d: value did not match, expect: %s, actual: %s", i, e.value, val)
		}
	}
}

func TestEntryReaderSimpleDict(t *testing.T) {
	var elems *starlark.Dict
	elems = &starlark.Dict{}
	elems.SetKey(starlark.String("a"), starlark.MakeInt(1))
	elems.SetKey(starlark.String("b"), starlark.MakeInt(2))
	elems.SetKey(starlark.String("c"), starlark.MakeInt(3))
	st := &dataset.Structure{
		Schema: dataset.BaseSchemaObject,
	}
	r := NewEntryReader(st, elems)

	expect := []struct {
		index int
		key   string
		value string
	}{
		{0, "a", "1"},
		{0, "b", "2"},
		{0, "c", "3"},
	}

	for i, e := range expect {
		ent, err := r.ReadEntry()
		if err != nil {
			log.Fatal(err)
		}

		if e.index != ent.Index {
			t.Errorf("case %d: index did not match, expect: %d, actual: %d", i, e.index, ent.Index)
		}
		if e.key != ent.Key {
			t.Errorf("case %d: key did not match, expect: %s, actual: %s", i, e.key, ent.Key)
		}
		val := fmt.Sprintf("%v", ent.Value)
		if e.value != val {
			t.Errorf("case %d: value did not match, expect: %s, actual: %s", i, e.value, val)
		}
	}
}
