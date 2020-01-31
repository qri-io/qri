package lib

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/rpc"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
	testrepo "github.com/qri-io/qri/repo/test"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestNewRenderRequests(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Errorf("expected NewRenderRequests to panic")
		}
	}()

	tr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte{}) }))
	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Errorf("error allocating listener: %s", err.Error())
		return
	}

	reqs := NewRenderRequests(tr, nil)
	if reqs.CoreRequestsName() != "render" {
		t.Errorf("invalid requests name. expected: '%s', got: '%s'", "render", reqs.CoreRequestsName())
	}

	// this should panic:
	NewRenderRequests(tr, rpc.NewClient(conn))
}

func TestRenderRequestsRender(t *testing.T) {
	// set Default Template to something easier to work with, then
	// cleanup when test completes
	prevDefaultTemplate := base.DefaultTemplate
	base.DefaultTemplate = `<html><h1>{{.Peername}}/{{.Name}}</h1></html>`
	defer func() { base.DefaultTemplate = prevDefaultTemplate }()

	cases := []struct {
		description string
		params      *RenderParams
		expect      []byte
		err         string
	}{
		{"no ref",
			&RenderParams{}, nil, "repo: empty dataset reference"},
		{"invalid ref",
			&RenderParams{
				Ref: "foo/invalid_ref",
			}, nil, "unknown dataset 'foo/invalid_ref'"},
		{"template override just title",
			&RenderParams{
				Ref:      "me/movies",
				Template: []byte("{{ .Meta.Title }}"),
			}, []byte("example movie data"), ""},
		{"override with invalid template",
			&RenderParams{
				Ref:      "me/movies",
				Template: []byte("{{ .BadTemplate }}"),
			}, nil, `template: index.html:1:3: executing "index.html" at <.BadTemplate>: can't evaluate field BadTemplate in type *dataset.Dataset`},
		{"override with corrupt template",
			&RenderParams{
				Ref:      "me/movies",
				Template: []byte("{{ .BadTemplateBooPlzFail"),
			}, nil, `parsing template: template: index.html:1: unclosed action`},
		{"default template",
			&RenderParams{
				Ref: "me/movies",
			}, []byte("<html><h1>peer/movies</h1></html>"), ""},
		{"alternate dataset default template",
			&RenderParams{
				Ref: "me/sitemap",
			}, []byte("<html><h1>peer/sitemap</h1></html>"), ""},
	}

	tr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	reqs := NewRenderRequests(tr, nil)

	for i, c := range cases {
		got := []byte{}
		err := reqs.RenderViz(c.params, &got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d %s error mismatch. expected: '%s', got: '%s'", i, c.description, c.err, err)
			return
		}

		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(string(got), string(c.expect), false)
		if len(diffs) > 1 {
			t.Log(dmp.DiffPrettyText(diffs))
			t.Errorf("case %d %s failed to match.", i, c.description)
		}
	}
}

// renderTestRunner holds state to make it easier to run tests
type renderTestRunner struct {
	Node        *p2p.QriNode
	Repo        repo.Repo
	DatasetReqs *DatasetRequests
	RenderReqs  *RenderRequests
	Context     context.Context
	ContextDone func()
	TsFunc      func() time.Time
}

// newRenderTestRunner returns a test runner for render
func newRenderTestRunner(t *testing.T, testName string) *renderTestRunner {
	r := renderTestRunner{}
	r.Context, r.ContextDone = context.WithCancel(context.Background())

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	r.TsFunc = dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Time{} }

	var err error
	r.Repo, err = testrepo.NewTestRepo()
	if err != nil {
		panic(err)
	}

	r.Node, err = p2p.NewQriNode(r.Repo, config.DefaultP2PForTesting())
	if err != nil {
		panic(err)
	}
	r.DatasetReqs = NewDatasetRequests(r.Node, nil)
	r.RenderReqs = NewRenderRequests(r.Repo, nil)

	return &r
}

// Delete cleans up after the test is done
func (r *renderTestRunner) Delete() {
	r.ContextDone()
	dsfs.Timestamp = r.TsFunc
}

// Save saves a version of the dataset with a body
func (r *renderTestRunner) Save(ref string, ds *dataset.Dataset, bodyPath string) {
	dsRef := reporef.DatasetRef{}
	params := SaveParams{
		Ref:      ref,
		Dataset:  ds,
		BodyPath: bodyPath,
	}
	err := r.DatasetReqs.Save(&params, &dsRef)
	if err != nil {
		panic(err)
	}
}

// Test that render with a readme returns an html string
func TestRenderReadme(t *testing.T) {
	runner := newRenderTestRunner(t, "render_readme")
	defer runner.Delete()

	runner.Save(
		"me/my_dataset",
		&dataset.Dataset{
			Readme: &dataset.Readme{
				ScriptBytes: []byte("# hi\n\nhello\n"),
			},
		},
		"testdata/jobs_by_automation/body.csv")

	params := RenderParams{
		Ref:       "me/my_dataset",
		OutFormat: "html",
	}
	var text string
	err := runner.RenderReqs.RenderReadme(&params, &text)
	if err != nil {
		t.Fatal(err)
	}

	expect := "<h1>hi</h1>\n\n<p>hello</p>\n"
	if diff := cmp.Diff(expect, text); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}
}
