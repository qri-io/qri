package ds

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/starlib/testdata"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarktest"
)

func callMethod(thread *starlark.Thread, v starlark.HasAttrs, name string, tuple starlark.Tuple) (starlark.Value, error) {
	method, err := v.Attr(name)
	if err != nil {
		return nil, err
	}
	if method == nil {
		return nil, fmt.Errorf("method %s does not exist", name)
	}
	return starlark.Call(thread, method, tuple, nil)
}

func TestCannotSetIfReadOnly(t *testing.T) {
	ds := NewDataset(&dataset.Dataset{})
	ds.Freeze()
	expect := "cannot set, Dataset is frozen"
	err := ds.SetField("body", starlark.NewList([]starlark.Value{starlark.NewList([]starlark.Value{starlark.String("a")})}))
	if err == nil {
		t.Fatal("expected error, did not get one")
	}
	if err.Error() != expect {
		t.Errorf("expected error: %s, got: %s", expect, err)
	}
}

func TestSetBody(t *testing.T) {
	ds := NewDataset(&dataset.Dataset{})
	err := ds.SetField("body", starlark.NewList([]starlark.Value{starlark.NewList([]starlark.Value{starlark.String("a")})}))
	if err != nil {
		t.Fatal(err)
	}
	expect := `     0
0    a
`
	bd, _ := ds.Attr("body")
	actual := bd.String()
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestChangeBody(t *testing.T) {
	t.Skip("TODO(dustmop): dataset.set_body is changing to dataset.body =")
	// Create the previous version with the body ["b"]
	prev := &dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	}
	prev.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[\"b\"]")))
	ds := NewDataset(prev)
	thread := &starlark.Thread{}

	body, err := callMethod(thread, ds, "get_body", starlark.Tuple{})
	if err != nil {
		t.Error(err)
	}
	expect := `["b"]`
	if fmt.Sprintf("%s", body) != expect {
		t.Errorf("expected body: %s, got: %s", expect, body)
	}

	_, err = callMethod(thread, ds, "set_body", starlark.Tuple{starlark.NewList([]starlark.Value{starlark.String("a")})})
	if err != nil {
		t.Error(err)
	}

	body, err = callMethod(thread, ds, "get_body", starlark.Tuple{})
	if err != nil {
		t.Error(err)
	}
	expect = `["a"]`
	if fmt.Sprintf("%s", body) != expect {
		t.Errorf("expected body: %s, got: %s", expect, body)
	}
}

func TestChangeBodyEvenIfTheSame(t *testing.T) {
	t.Skip("TODO(dustmop): dataset.set_body is changing to dataset.body =")
	// Create the previous version with the body ["a"]
	prev := &dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	}
	prev.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[\"a\"]")))
	ds := NewDataset(prev)
	thread := &starlark.Thread{}

	body, err := callMethod(thread, ds, "get_body", starlark.Tuple{})
	if err != nil {
		t.Error(err)
	}
	expect := `["a"]`
	if fmt.Sprintf("%s", body) != expect {
		t.Errorf("expected body: %s, got: %s", expect, body)
	}

	_, err = callMethod(thread, ds, "set_body", starlark.Tuple{starlark.NewList([]starlark.Value{starlark.String("a")})})
	if err != nil {
		t.Error(err)
	}

	body, err = callMethod(thread, ds, "get_body", starlark.Tuple{})
	if err != nil {
		t.Error(err)
	}
	expect = `["a"]`
	if fmt.Sprintf("%s", body) != expect {
		t.Errorf("expected body: %s, got: %s", expect, body)
	}
}

func TestFile(t *testing.T) {
	resolve.AllowFloat = true
	thread := &starlark.Thread{Load: newLoader()}
	starlarktest.SetReporter(thread, t)

	// Execute test file
	_, err := starlark.ExecFile(thread, "testdata/test.star", nil, starlark.StringDict{
		"csv_ds": csvDataset(),
	})
	if err != nil {
		if ee, ok := err.(*starlark.EvalError); ok {
			t.Error(ee.Backtrace())
		} else {
			t.Error(err)
		}
	}
}

// load implements the 'load' operation as used in the evaluator tests.
func newLoader() func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	return testdata.NewLoader(LoadModule, ModuleName)
}

func csvDataset() *Dataset {
	text := `title,count,is great
foo,1,true
bar,2,false
bat,3,meh
`
	ds := &dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "csv",
			FormatConfig: map[string]interface{}{
				"headerRow": true,
			},
			Schema: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "array",
					"items": []interface{}{
						map[string]interface{}{"title": "title", "type": "string"},
						map[string]interface{}{"title": "count", "type": "integer"},
						map[string]interface{}{"title": "is great", "type": "string"},
					},
				},
			},
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.csv", []byte(text)))

	d := NewDataset(ds)
	return d
}
