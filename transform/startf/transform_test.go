package startf

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/stepfile"
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

	if err := ExecScript(ctx, ds); err != nil {
		t.Fatal(err)
	}
	if ds.Transform == nil {
		t.Error("expected transform")
	}

	entryReader, err := dsio.NewEntryReader(ds.Structure, ds.BodyFile())
	if err != nil {
		t.Fatalf("couldn't create entry reader from returned dataset & body file: %s", err.Error())
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
	if ds.Structure.Entries != 3 {
		t.Errorf("expected `ds.Structure.Entries` to be 3, got: %d", i)
	}
}

func TestExecScript2(t *testing.T) {
	ctx := context.Background()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"foo":[["bar","baz","bat"]]}`))
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
				{
					Name:     "transform",
					Syntax:   "starlark",
					Category: "transform",
					Script:   "ds = dataset.latest()\nds.body = [[1,2,3]]\ndataset.commit(ds)",
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
	// Check that the `ds.Structure.Entries` was set
	if ds.Structure == nil {
		t.Fatal("dataset does not have a structure assigned")
	}
	if ds.Structure.Entries != 1 {
		t.Errorf("`ds.Structure.Entries` not assigned correctly, expected 1, got: %d", ds.Structure.Entries)
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

func TestEditMeta(t *testing.T) {
	ctx := context.Background()
	r := testRepo(t)
	ds := &dataset.Dataset{
		Peername:  "peer",
		Name:      "movies",
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/set_meta.star"))

	err := ExecScript(ctx, ds, func(o *ExecOpts) {
		o.ModuleLoader = testModuleLoader(t)
		o.DatasetLoader = base.NewTestDatasetLoader(r.Filesystem(), r)
	})
	if err != nil {
		t.Fatal(err)
	}
	if ds.Meta.Title != "new title" {
		t.Errorf("meta title was not changed")
	}
	if ds.Structure.Entries != 2335 {
		t.Errorf("`ds.Structure.Entries` incorrect, expected 2335, got: %d", ds.Structure.Entries)
	}
}

func TestScriptError(t *testing.T) {
	ctx := context.Background()
	script := `error("script error")`
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

func ExecScript(ctx context.Context, ds *dataset.Dataset, opts ...func(o *ExecOpts)) error {
	// Convert single-file transform scripts to steps
	if len(ds.Transform.Steps) == 0 && ds.Transform.ScriptFile() != nil {
		steps, err := stepfile.Read(ds.Transform.ScriptFile())
		if err != nil {
			return err
		}
		ds.Transform.Steps = steps
	}

	runner := NewStepRunner(ds, opts...)
	for _, step := range ds.Transform.Steps {
		if err := runner.RunStep(ctx, ds, step); err != nil {
			return err
		}
	}
	return nil
}
