package base

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/tabular"
	"github.com/qri-io/qfs"
)

func TestGetBody(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	ds, err := ReadDataset(ctx, r, ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err = OpenDataset(ctx, r.Filesystem(), ds); err != nil {
		t.Fatal(err)
	}

	gotBody, err := GetBody(ds, 1, 1, false)
	if err != nil {
		t.Error(err.Error())
	}
	gotBodyBytes, err := json.Marshal(gotBody)
	if err != nil {
		t.Errorf(err.Error())
	}
	expectBodyBytes := []byte(`[["new york",8500000,44.4,true]]`)
	if diff := cmp.Diff(expectBodyBytes, gotBodyBytes); diff != "" {
		t.Errorf("GetBody output (-want +got):\n%s", diff)
	}
}

func TestReadBodyBytes(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	ds, err := ReadDataset(ctx, r, ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err = OpenDataset(ctx, r.Filesystem(), ds); err != nil {
		t.Fatal(err)
	}

	data, err := ReadBodyBytes(ds, dataset.JSONDataFormat, nil, 1, 1, false)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(data, []byte(`[["new york",8500000,44.4,true]]`)) {
		t.Errorf("byte response mismatch. got: %s", string(data))
	}

	expectPath := "/mem/QmcCcPTqmckdXLBwPQXxfyW2BbFcUT6gqv9oGeWDkrNTyD"
	if expectPath != ds.BodyPath {
		t.Errorf("bodypath mismatch. want %q got %q", expectPath, ds.BodyPath)
	}
}

func TestConvertBodyFormat(t *testing.T) {
	jsonStructure := &dataset.Structure{Format: "json", Schema: dataset.BaseSchemaArray}
	csvStructure := &dataset.Structure{Format: "csv", Schema: tabular.BaseTabularSchema}

	// CSV -> JSON
	body := qfs.NewMemfileBytes("", []byte("a,b,c"))
	got, err := ConvertBodyFormat(body, csvStructure, jsonStructure)
	if err != nil {
		t.Error(err.Error())
	}
	data, err := ioutil.ReadAll(got)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(data, []byte(`[["a","b","c"]]`)) {
		t.Error(fmt.Errorf("converted body didn't match, got: %s", data))
	}

	// CSV -> JSON, multiple lines
	body = qfs.NewMemfileBytes("", []byte("a,b,c\n\rd,e,f\n\rg,h,i"))
	got, err = ConvertBodyFormat(body, csvStructure, jsonStructure)
	if err != nil {
		t.Fatal(err.Error())
	}
	data, err = ioutil.ReadAll(got)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(data, []byte(`[["a","b","c"],["d","e","f"],["g","h","i"]]`)) {
		t.Error(fmt.Errorf("converted body didn't match, got: %s", data))
	}

	// JSON -> CSV
	body = qfs.NewMemfileBytes("", []byte(`[["a","b","c"]]`))
	got, err = ConvertBodyFormat(body, jsonStructure, csvStructure)
	if err != nil {
		t.Fatal(err.Error())
	}
	data, err = ioutil.ReadAll(got)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(data, []byte("a,b,c\n")) {
		t.Error(fmt.Errorf("converted body didn't match, got: %s", data))
	}

	// CSV -> CSV
	body = qfs.NewMemfileBytes("", []byte("a,b,c"))
	got, err = ConvertBodyFormat(body, csvStructure, csvStructure)
	if err != nil {
		t.Fatal(err.Error())
	}
	data, err = ioutil.ReadAll(got)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(data, []byte("a,b,c\n")) {
		t.Error(fmt.Errorf("converted body didn't match, got: %s", data))
	}

	// JSON -> JSON
	body = qfs.NewMemfileBytes("", []byte(`[["a","b","c"]]`))
	got, err = ConvertBodyFormat(body, jsonStructure, jsonStructure)
	if err != nil {
		t.Fatal(err.Error())
	}
	data, err = ioutil.ReadAll(got)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(data, []byte(`[["a","b","c"]]`)) {
		t.Error(fmt.Errorf("converted body didn't match, got: %s", data))
	}
}

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
