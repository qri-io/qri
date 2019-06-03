package ds

import (
	"fmt"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/starlib/testdata"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarktest"
)

func TestCheckFields(t *testing.T) {
	fieldErr := fmt.Errorf("can't mutate this field")
	allErrCheck := func(fields ...string) error {
		return fieldErr
	}
	ds := NewDataset(nil, allErrCheck)
	ds.SetMutable(&dataset.Dataset{})
	thread := &starlark.Thread{}

	if _, err := ds.SetBody(thread, nil, starlark.Tuple{starlark.String("data")}, nil); err != fieldErr {
		t.Errorf("expected fieldErr, got: %s", err)
	}

	if _, err := ds.SetMeta(thread, nil, starlark.Tuple{starlark.String("key"), starlark.String("value")}, nil); err != fieldErr {
		t.Errorf("expected fieldErr, got: %s", err)
	}

	if _, err := ds.SetStructure(thread, nil, starlark.Tuple{starlark.String("wut")}, nil); err != fieldErr {
		t.Errorf("expected fieldErr, got: %s", err)
	}
}

func TestCannotSetIfReadOnly(t *testing.T) {
	ds := NewDataset(&dataset.Dataset{}, nil)
	thread := &starlark.Thread{}
	expect := "cannot call set_body on read-only dataset"
	_, err := ds.SetBody(thread, nil, starlark.Tuple{starlark.NewList([]starlark.Value{starlark.String("a")})}, nil)
	if err.Error() != expect {
		t.Errorf("expected error: %s, got: %s", expect, err)
	}
	if ds.IsBodyModified() {
		t.Errorf("expected body to not have been modified")
	}
}

func TestSetMutable(t *testing.T) {
	ds := NewDataset(&dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	}, nil)
	ds.SetMutable(&dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	})
	thread := &starlark.Thread{}

	_, err := ds.SetBody(thread, nil, starlark.Tuple{starlark.NewList([]starlark.Value{starlark.String("a")})}, nil)
	if err != nil {
		t.Error(err)
	}
	if !ds.IsBodyModified() {
		t.Errorf("expected body to have been modified")
	}

	body, err := ds.GetBody(thread, nil, starlark.Tuple{}, nil)
	if err != nil {
		t.Error(err)
	}
	expect := `["a"]`
	if fmt.Sprintf("%s", body) != expect {
		t.Errorf("expected body: %s, got: %s", expect, body)
	}
}

func TestChangeBody(t *testing.T) {
	// Create the previous version with the body ["b"]
	prev := &dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	}
	prev.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[\"b\"]")))
	ds := NewDataset(prev, nil)
	// Next version has no body yet
	ds.SetMutable(&dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	})
	thread := &starlark.Thread{}

	body, err := ds.GetBody(thread, nil, starlark.Tuple{}, nil)
	if err != nil {
		t.Error(err)
	}
	expect := `["b"]`
	if fmt.Sprintf("%s", body) != expect {
		t.Errorf("expected body: %s, got: %s", expect, body)
	}

	_, err = ds.SetBody(thread, nil, starlark.Tuple{starlark.NewList([]starlark.Value{starlark.String("a")})}, nil)
	if err != nil {
		t.Error(err)
	}
	if !ds.IsBodyModified() {
		t.Errorf("expected body to have been modified")
	}

	body, err = ds.GetBody(thread, nil, starlark.Tuple{}, nil)
	if err != nil {
		t.Error(err)
	}
	expect = `["a"]`
	if fmt.Sprintf("%s", body) != expect {
		t.Errorf("expected body: %s, got: %s", expect, body)
	}
}

func TestChangeBodyEvenIfTheSame(t *testing.T) {
	// Create the previous version with the body ["a"]
	prev := &dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	}
	prev.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[\"a\"]")))
	ds := NewDataset(prev, nil)
	// Next version has no body yet
	ds.SetMutable(&dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	})
	thread := &starlark.Thread{}

	body, err := ds.GetBody(thread, nil, starlark.Tuple{}, nil)
	if err != nil {
		t.Error(err)
	}
	expect := `["a"]`
	if fmt.Sprintf("%s", body) != expect {
		t.Errorf("expected body: %s, got: %s", expect, body)
	}

	_, err = ds.SetBody(thread, nil, starlark.Tuple{starlark.NewList([]starlark.Value{starlark.String("a")})}, nil)
	if err != nil {
		t.Error(err)
	}
	if !ds.IsBodyModified() {
		t.Errorf("expected body to have been modified")
	}

	body, err = ds.GetBody(thread, nil, starlark.Tuple{}, nil)
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
		"csv_ds": csvDataset().Methods(),
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

	d := NewDataset(ds, nil)
	d.SetMutable(&dataset.Dataset{
		Structure: ds.Structure,
	})
	return d
}
