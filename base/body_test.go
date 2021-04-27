package base

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
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
