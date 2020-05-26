package startf

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
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
	AddMutateFieldCheck(nil)(o)

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
	err := ExecScript(ctx, ds, nil, SetErrWriter(stderr))
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
	expect := `🤖  running transform...
hello world!`
	if string(output) != expect {
		t.Errorf("stderr mismatch. expected: '%s', got: '%s'", expect, string(output))
	}

	entryReader, err := dsio.NewEntryReader(ds.Structure, ds.BodyFile())
	if err != nil {
		t.Errorf("couldn't create entry reader from returned dataset & body file: %s", err.Error())
		return
	}

	i := 0
	dsio.EachEntry(entryReader, func(_ int, x dsio.Entry, e error) error {
		if e != nil {
			t.Errorf("entry %d iteration error: %s", i, e.Error())
		}
		i++
		return nil
	})

	if i != 8 {
		t.Errorf("expected 8 entries, got: %d", i)
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
	err := ExecScript(ctx, ds, nil, func(o *ExecOpts) {
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
	if err := ExecScript(ctx, ds, nil); err == nil {
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

	err := ExecScript(ctx, ds, nil, func(o *ExecOpts) {
		o.Repo = r
		o.ModuleLoader = testModuleLoader(t)
		o.DatasetLoader = dsref.NewParseResolveLoadFunc("", r, repoLoader{r})
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TODO(b5) - we should think about moving this somewhere more general
type repoLoader struct {
	r repo.Repo
}

func (rl repoLoader) LoadDataset(ctx context.Context, ref dsref.Ref, source string) (*dataset.Dataset, error) {
	var (
		ds  *dataset.Dataset
		err error
	)

	if ds, err = dsfs.LoadDataset(ctx, rl.r.Store(), ref.Path); err != nil {
		return nil, err
	}
	// Set transient info on the returned dataset
	ds.Name = ref.Name
	ds.Peername = ref.Username

	// TODO (b5) - this should be a call to base.OpenDatasets
	if ds.BodyFile() == nil {
		if err = ds.OpenBodyFile(ctx, rl.r.Filesystem()); err != nil {
			return nil, err
		}
	}

	return ds, nil
}

func TestGetMetaNilPrev(t *testing.T) {
	ctx := context.Background()
	ds := &dataset.Dataset{
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/meta_title.star"))
	err := ExecScript(ctx, ds, nil)
	if err != nil {
		t.Error(err.Error())
		return
	}
	data, _ := ioutil.ReadAll(ds.BodyFile())
	actual := string(data)
	expect := `["no title"]`
	if actual != expect {
		t.Errorf("expected: \"%s\", actual: \"%s\"", expect, actual)
	}
}

func TestGetMetaWithPrev(t *testing.T) {
	ctx := context.Background()
	ds := &dataset.Dataset{
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/meta_title.star"))
	prev := &dataset.Dataset{
		Meta: &dataset.Meta{
			Title: "test_title",
		},
	}
	err := ExecScript(ctx, ds, prev)
	if err != nil {
		t.Error(err.Error())
		return
	}
	data, _ := ioutil.ReadAll(ds.BodyFile())
	actual := string(data)
	expect := `["title: test_title"]`
	if actual != expect {
		t.Errorf("expected: \"%s\", actual: \"%s\"", expect, actual)
	}
}

func TestMutatedComponentsFunc(t *testing.T) {
	ds := &dataset.Dataset{
		Commit:    &dataset.Commit{},
		Meta:      &dataset.Meta{},
		Transform: &dataset.Transform{},
		Structure: &dataset.Structure{},
		Viz:       &dataset.Viz{},
		Body:      []interface{}{"foo"},
	}

	fn := MutatedComponentsFunc(ds)

	paths := []string{
		"commit",
		"meta",
		"transform",
		"structure",
		"viz",
		"body",
	}
	for _, p := range paths {
		if err := fn(p); err == nil {
			t.Errorf("expected error for path: '%s', got nil", p)
		}
	}

	fn = MutatedComponentsFunc(&dataset.Dataset{})
	for _, p := range paths {
		if err := fn(p); err != nil {
			t.Errorf("expected empty dataset to not error for path: '%s', got: %s", p, err)
		}
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
