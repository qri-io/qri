package startf

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/starlib/testdata"
	repoTest "github.com/qri-io/qri/repo/test"
	"github.com/qri-io/starlib"
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

func TestExecScript(t *testing.T) {
	ds := &dataset.Dataset{
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/tf.star"))

	stderr := &bytes.Buffer{}
	err := ExecScript(ds, nil, SetOutWriter(stderr))
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

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"foo":["bar","baz","bat"]}`))
	}))

	ds := &dataset.Dataset{
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/fetch.star"))
	err := ExecScript(ds, nil, func(o *ExecOpts) {
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

func TestLoadDataset(t *testing.T) {
	node := testQriNode(t)

	ds := &dataset.Dataset{
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/load_ds.star"))

	err := ExecScript(ds, nil, func(o *ExecOpts) {
		o.Node = node
		o.ModuleLoader = testModuleLoader(t)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetMetaNilPrev(t *testing.T) {
	ds := &dataset.Dataset{
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/meta_title.star"))
	err := ExecScript(ds, nil)
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
	ds := &dataset.Dataset{
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/meta_title.star"))
	prev := &dataset.Dataset{
		Meta: &dataset.Meta{
			Title: "test_title",
		},
	}
	err := ExecScript(ds, prev)
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

func testQriNode(t *testing.T) *p2p.QriNode {
	mr, err := repoTest.NewTestRepo(nil)
	if err != nil {
		t.Fatal(err)
	}

	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}
	return node
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
