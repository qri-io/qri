package startf

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/repo"
	repoTest "github.com/qri-io/qri/repo/test"
	"github.com/qri-io/starlib"
	"github.com/qri-io/starlib/testdata"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarktest"
)

func scriptFile(t *testing.T, path string) qfs.File {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return qfs.NewMemfileBytes(path, data)
}

func TestOpts(t *testing.T) {
	o := &ExecOpts{}
	SetSecrets(nil)(o)
	SetSecrets(map[string]string{"a": "b"})(o)
	SetErrWriter(nil)(o)
	AddQriRepo(nil)(o)

	expect := &ExecOpts{
		Secrets:   map[string]interface{}{"a": "b"},
		ErrWriter: nil,
	}

	if diff := cmp.Diff(expect, o); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestExecScript(t *testing.T) {
	ctx := context.Background()
	ds := &dataset.Dataset{
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/tf.star"))

	stderr := &bytes.Buffer{}
	err := ExecScript(ctx, ds, SetErrWriter(stderr))
	if err != nil {
		t.Error(err.Error())
		return
	}
	if ds.Transform == nil {
		t.Error("expected transform")
	}

	output, err := ioutil.ReadAll(stderr)
	if err != nil {
		t.Fatal(err)
	}
	expect := "hello world!\n"
	if string(output) != expect {
		t.Errorf("stderr mismatch. expected: '%s', got: '%s'", expect, string(output))
	}

	bf := ds.BodyFile()
	if bf == nil {
		t.Fatal("body file is nil")
	}
	entryReader, err := dsio.NewEntryReader(ds.Structure, bf)
	if err != nil {
		t.Errorf("couldn't create entry reader from returned dataset & body file: %s", err.Error())
		return
	}

	i := 0
	dsio.EachEntry(entryReader, func(n int, x dsio.Entry, e error) error {
		if e != nil {
			t.Errorf("entry %d iteration error: %s", i, e.Error())
		}
		i++
		return nil
	})

	if i != 3 {
		t.Errorf("expected 3 entries, got: %d", i)
	}
}

func TestExecScript2(t *testing.T) {
	ctx := context.Background()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"foo":["bar","baz","bat"]}`))
	}))

	ds := &dataset.Dataset{
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/fetch.star"))
	err := ExecScript(ctx, ds, func(o *ExecOpts) {
		o.Globals["test_server_url"] = starlark.String(s.URL)
	})

	if err != nil {
		t.Error(err.Error())
		return
	}
	if ds.Transform == nil {
		t.Error("expected transform")
	}
}

func TestExecStep(t *testing.T) {
	ctx := context.Background()
	ds := &dataset.Dataset{
		Transform: &dataset.Transform{
			Steps: []*dataset.TransformStep{
				&dataset.TransformStep{
					Name:     "transform",
					Syntax:   "starlark",
					Category: "transform",
					Script:   "def transform(ds, ctx):\n  ds.body = [[1,2,3]]",
				},
			},
		},
	}
	// Run the single step.
	stepRunner := NewStepRunner(ds)
	err := stepRunner.RunStep(ctx, ds, ds.Transform.Steps[0])
	if err != nil {
		t.Fatal(err)
	}
	// Check that body was set by the transform step.
	bodyfile := ds.BodyFile()
	if bodyfile == nil {
		t.Fatal("dataset did not have body assigned")
	}
	data, err := ioutil.ReadAll(bodyfile)
	if err != nil {
		t.Fatal(err)
	}
	actual := string(data)
	expect := `[[1,2,3]]`
	if actual != expect {
		t.Errorf("expected: %q, actual: %q", expect, actual)
	}
}

func TestScriptError(t *testing.T) {
	ctx := context.Background()
	script := `
def transform(ds, ctx):
	error("script error")
	`
	scriptFile := qfs.NewMemfileBytes("tf.star", []byte(script))

	ds := &dataset.Dataset{
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile)
	if err := ExecScript(ctx, ds); err == nil {
		t.Errorf("expected script to error. got nil")
	}
}

func TestLoadDataset(t *testing.T) {
	ctx := context.Background()
	r := testRepo(t)

	ds := &dataset.Dataset{
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/load_ds.star"))

	err := ExecScript(ctx, ds, func(o *ExecOpts) {
		o.Repo = r
		o.ModuleLoader = testModuleLoader(t)
		o.DatasetLoader = base.NewTestDatasetLoader(r.Filesystem(), r)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetMetaNilPrev(t *testing.T) {
	ctx := context.Background()
	ds := &dataset.Dataset{
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/meta_title.star"))
	err := ExecScript(ctx, ds)
	if err != nil {
		t.Error(err.Error())
		return
	}
	bodyfile := ds.BodyFile()
	if bodyfile == nil {
		t.Fatal("dataset did not have body assigned")
	}
	data, _ := ioutil.ReadAll(bodyfile)
	actual := string(data)
	expect := `[["no title"]]`
	if actual != expect {
		t.Errorf("expected: \"%s\", actual: \"%s\"", expect, actual)
	}
}

func TestGetMetaWithPrev(t *testing.T) {
	ctx := context.Background()
	ds := &dataset.Dataset{
		Meta: &dataset.Meta{
			Title: "test_title",
		},
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/meta_title.star"))
	err := ExecScript(ctx, ds)
	if err != nil {
		t.Error(err.Error())
		return
	}
	bodyfile := ds.BodyFile()
	if bodyfile == nil {
		t.Fatal("dataset did not have body assigned")
	}
	data, _ := ioutil.ReadAll(bodyfile)
	actual := string(data)
	expect := `[["title: test_title"]]`
	if actual != expect {
		t.Errorf("expected: \"%s\", actual: \"%s\"", expect, actual)
	}
}

func testRepo(t *testing.T) repo.Repo {
	mr, err := repoTest.NewTestRepo()
	if err != nil {
		t.Fatal(err)
	}
	return mr
}

func testModuleLoader(t *testing.T) func(thread *starlark.Thread, module string) (dict starlark.StringDict, err error) {
	assertLoader := testdata.NewLoader(nil, "")
	return func(thread *starlark.Thread, module string) (dict starlark.StringDict, err error) {
		starlarktest.SetReporter(thread, t)
		if module == "assert.star" {
			return assertLoader(thread, module)
		}
		return starlib.Loader(thread, module)
	}
}
