package context

import (
	"testing"

	"github.com/qri-io/starlib/testdata"
	"go.starlark.net/starlark"
)

func TestContext(t *testing.T) {
	thread := &starlark.Thread{Load: newLoader()}

	dlCtx := NewContext(nil, nil)
	dlCtx.SetResult("download", starlark.Bool(true))

	// Execute test file
	_, err := starlark.ExecFile(thread, "testdata/test.star", nil, starlark.StringDict{
		"ctx": NewContext(
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{"baz": "bat"},
		).Struct(),
		"dl_ctx": dlCtx.Struct(),
	})
	if err != nil {
		t.Error(err)
	}
}

func TestMissingValue(t *testing.T) {
	thread := &starlark.Thread{}

	ctx := NewContext(nil, nil)
	val, err := ctx.getValue(thread, nil, starlark.Tuple{starlark.String("foo")}, nil)
	if val != starlark.None {
		t.Errorf("expected none return value")
	}

	expect := "value foo not set in context"
	if err.Error() != expect {
		t.Errorf("error message mismatch. expected: %s, got: %s", expect, err.Error())
	}
}

func TestMissingConfig(t *testing.T) {
	thread := &starlark.Thread{}
	ctx := NewContext(nil, nil)

	val, err := ctx.GetConfig(thread, nil, starlark.Tuple{starlark.String("foo")}, nil)
	if val != starlark.None {
		t.Errorf("expected none return value")
	}

	expect := "no config provided"
	if err.Error() != expect {
		t.Errorf("error message mismatch. expected: %s, got: %s", expect, err.Error())
	}
}

func TestMissingSecrets(t *testing.T) {
	thread := &starlark.Thread{}
	ctx := NewContext(nil, nil)

	val, err := ctx.GetSecret(thread, nil, starlark.Tuple{starlark.String("foo")}, nil)
	if val != starlark.None {
		t.Errorf("expected none return value")
	}

	expect := "no secrets provided"
	if err.Error() != expect {
		t.Errorf("error message mismatch. expected: %s, got: %s", expect, err.Error())
	}
}

// load implements the 'load' operation as used in the evaluator tests.
func newLoader() func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	return testdata.NewLoader(nil, "context_is_global_no_module_name_exists")
}
