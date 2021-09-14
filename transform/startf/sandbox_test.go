package startf

import (
	"testing"

	"github.com/qri-io/starlib/testdata"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarktest"
)

func TestConfig(t *testing.T) {
	cfg := config(map[string]interface{}{
		"a": false,
		"b": 1,
		"c": "string",
	})

	thread := &starlark.Thread{Load: testdata.NewLoader(nil, "")}
	starlarktest.SetReporter(thread, t)
	src := `
load("assert.star", "assert")
assert.eq(config.get("foo"), None)
assert.eq(config.get("foo", 5), 5)
assert.eq(config.get("a"), False)
assert.eq(config.get("b"), 1)
assert.eq(config.get("c"), "string")
`
	_, err := starlark.ExecFile(thread, "test_config.star", src, starlark.StringDict{"config": cfg})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSecrets(t *testing.T) {
	s := secrets(map[string]interface{}{
		"a": false,
		"b": 1,
		"c": "string",
	})

	thread := &starlark.Thread{Load: testdata.NewLoader(nil, "")}
	starlarktest.SetReporter(thread, t)
	src := `
load("assert.star", "assert")
assert.eq(secrets.get("foo"), None)
assert.eq(secrets.get("foo", 5), 5)
assert.eq(secrets.get("a"), False)
assert.eq(secrets.get("b"), 1)
assert.eq(secrets.get("c"), "string")
`
	_, err := starlark.ExecFile(thread, "test_config.star", src, starlark.StringDict{"secrets": s})
	if err != nil {
		t.Fatal(err)
	}
}
