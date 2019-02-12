package base

import (
	"fmt"
	"strings"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
)

func TestReadEntriesArray(t *testing.T) {
	cases := []struct {
		description           string
		rdrCount, expectCount int
	}{
		{"read all 50", 50, 50},
	}

	for i, c := range cases {
		r := newTestJSONArrayReader(c.rdrCount)
		got, err := ReadEntries(r)
		if err != nil {
			t.Errorf("case %d %s unexpected error. '%s'", i, c.description, err)
			continue
		}
		arr, ok := got.([]interface{})
		if !ok {
			t.Errorf("case %d %s didn't return a []interface{}", i, c.description)
			continue
		}
		if len(arr) != c.expectCount {
			t.Errorf("case %d %s unexpected entry count. expected: %d got: %d", i, c.description, c.expectCount, len(arr))
		}
	}
}

func TestReadEntriesObject(t *testing.T) {
	cases := []struct {
		description           string
		rdrCount, expectCount int
	}{
		{"read all 50", 50, 50},
	}

	for i, c := range cases {
		r := newTestJSONObjectReader(c.rdrCount)
		got, err := ReadEntries(r)
		if err != nil {
			t.Errorf("case %d %s unexpected error. '%s'", i, c.description, err)
			continue
		}

		obj, ok := got.(map[string]interface{})
		if !ok {
			t.Errorf("case %d %s didn't return a []interface{}", i, c.description)
			continue
		}
		if len(obj) != c.expectCount {
			t.Errorf("case %d %s unexpected entry count. expected: %d got: %d", i, c.description, c.expectCount, len(obj))
		}
	}
}

// newTestJSONArrayReader creates a dsio.EntryReader with a number of entries that matches entryCount
// with an array as the top level type
func newTestJSONArrayReader(entryCount int) dsio.EntryReader {
	buf := &strings.Builder{}
	buf.WriteByte('[')
	for i := 0; i < entryCount; i++ {
		buf.WriteString(fmt.Sprintf(`{"id":%d}`, i))
		if i != entryCount-1 {
			buf.WriteByte(',')
		}
	}
	buf.WriteByte(']')
	st := &dataset.Structure{
		Format: "json",
		Schema: dataset.BaseSchemaArray,
	}
	er, err := dsio.NewJSONReader(st, strings.NewReader(buf.String()))
	if err != nil {
		panic(err)
	}
	return er
}

// newTestJSONArrayReader creates a dsio.EntryReader with a number of entries that matches entryCount
// using an object as a top level type
func newTestJSONObjectReader(entryCount int) dsio.EntryReader {
	buf := &strings.Builder{}
	buf.WriteByte('{')
	for i := 0; i < entryCount; i++ {
		buf.WriteString(fmt.Sprintf(`"entry_%d":{"id":%d}`, i, i))
		if i != entryCount-1 {
			buf.WriteByte(',')
		}
	}
	buf.WriteByte('}')
	st := &dataset.Structure{
		Format: "json",
		Schema: dataset.BaseSchemaObject,
	}
	er, err := dsio.NewJSONReader(st, strings.NewReader(buf.String()))
	if err != nil {
		panic(err)
	}
	return er
}
