package qri

import (
	"fmt"
	"testing"

	"github.com/qri-io/dataset"
	"go.starlark.net/starlark"
)

func TestNewModule(t *testing.T) {
	t.Skip("TODO (b5): restore")

	ds := &dataset.Dataset{
		Transform: &dataset.Transform{
			Syntax: "starlark",
			Config: map[string]interface{}{
				"foo": "bar",
			},
		},
	}

	thread := &starlark.Thread{Load: newLoader(ds)}

	// Execute test file
	_, err := starlark.ExecFile(thread, "testdata/test.star", nil, nil)
	if err != nil {
		t.Error(err)
	}
}

// load implements the 'load' operation as used in the evaluator tests.
func newLoader(ds *dataset.Dataset) func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	return func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
		if module == ModuleName {
			return starlark.StringDict{"qri": NewModule(nil).Struct()}, nil
		}

		return nil, fmt.Errorf("invalid module")
	}
}
