package lib

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestRenderMethodsRender(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

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
			&RenderParams{}, nil, "empty reference"},
		{"invalid ref",
			&RenderParams{
				Ref: "foo/invalid_ref",
			}, nil, "reference not found"},
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
	node, err := p2p.NewQriNode(tr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)
	rm := NewRenderMethods(inst)

	for i, c := range cases {
		got, err := rm.RenderViz(ctx, c.params)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d %s error mismatch. expected: '%s', got: '%s'", i, c.description, c.err, err)
			return
		}

		if diff := cmp.Diff(string(got), string(c.expect)); diff != "" {
			t.Errorf("case %d result mismatch. (-want +got):\n%s", i, c.description)
		}
	}
}

// renderTestRunner holds state to make it easier to run tests
type renderTestRunner struct {
	Node          *p2p.QriNode
	Repo          repo.Repo
	DatasetReqs   DatasetMethods
	RenderMethods *RenderMethods
	Context       context.Context
	ContextDone   func()
	TsFunc        func() time.Time
}

// newRenderTestRunner returns a test runner for render
func newRenderTestRunner(t *testing.T, testName string) *renderTestRunner {
	ctx, done := context.WithCancel(context.Background())
	defer done()

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

	r.Node, err = p2p.NewQriNode(r.Repo, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		panic(err)
	}
	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), r.Node)
	r.DatasetReqs = inst.Dataset()
	r.RenderMethods = NewRenderMethods(inst)

	return &r
}

// Delete cleans up after the test is done
func (r *renderTestRunner) Delete() {
	r.ContextDone()
	dsfs.Timestamp = r.TsFunc
}

// Save saves a version of the dataset with a body
func (r *renderTestRunner) Save(ref string, ds *dataset.Dataset, bodyPath string) {
	params := SaveParams{
		Ref:      ref,
		Dataset:  ds,
		BodyPath: bodyPath,
	}
	_, err := r.DatasetReqs.Save(r.Context, &params)
	if err != nil {
		panic(err)
	}
}

func TestRenderMethodsRenderViz(t *testing.T) {
	runner := newRenderTestRunner(t, "render_viz")
	defer runner.Delete()

	ctx := context.TODO()

	params := RenderParams{
		Ref: "foo/bar",
		Dataset: &dataset.Dataset{
			Readme: &dataset.Readme{
				ScriptBytes: []byte("# hi\n\nhello"),
			},
		},
	}
	if _, err := runner.RenderMethods.RenderViz(ctx, &params); err == nil {
		t.Errorf("expected RenderReadme with both ref & dataset to error")
	}

	params = RenderParams{
		Dataset: &dataset.Dataset{
			Readme: &dataset.Readme{
				ScriptBytes: []byte("# hi\n\nhello"),
			},
		},
	}
	if _, err := runner.RenderMethods.RenderViz(ctx, &params); err != nil {
		t.Error(err)
	}
}

// Test that render with a readme returns an html string
func TestRenderReadme(t *testing.T) {
	runner := newRenderTestRunner(t, "render_readme")
	defer runner.Delete()

	ctx := context.TODO()

	runner.Save(
		"me/my_dataset",
		&dataset.Dataset{
			Readme: &dataset.Readme{
				ScriptBytes: []byte("# hi\n\nhello\n"),
			},
		},
		"testdata/jobs_by_automation/body.csv")

	params := RenderParams{
		Ref:    "peer/my_dataset",
		Format: "html",
	}
	text, err := runner.RenderMethods.RenderReadme(ctx, &params)
	if err != nil {
		t.Fatal(err)
	}

	expect := "<h1>hi</h1>\n\n<p>hello</p>\n"
	if diff := cmp.Diff(expect, string(text)); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}

	params = RenderParams{
		Dataset: &dataset.Dataset{
			Readme: &dataset.Readme{
				ScriptBytes: []byte("# hi\n\nhello"),
			},
		},
	}
	text, err = runner.RenderMethods.RenderReadme(ctx, &params)
	if err != nil {
		t.Errorf("dynamic dataset render error: %s", err)
	}

	if diff := cmp.Diff(expect, string(text)); diff != "" {
		t.Errorf("dynamic dataset render response mismatch (-want +got):\n%s", diff)
	}

	params = RenderParams{
		Ref: "foo/bar",
		Dataset: &dataset.Dataset{
			Readme: &dataset.Readme{
				ScriptBytes: []byte("# hi\n\nhello"),
			},
		},
	}
	text, err = runner.RenderMethods.RenderReadme(ctx, &params)
	if err == nil {
		t.Errorf("expected RenderReadme with both ref & dataset to error")
	}
}

func TestRenderValidationFailure(t *testing.T) {
	runner := newRenderTestRunner(t, "render_readme")
	defer runner.Delete()

	params := RenderParams{
		Ref:     "peer/my_dataset",
		Dataset: &dataset.Dataset{},
		Format:  "html",
	}
	_, err := runner.RenderMethods.RenderReadme(runner.Context, &params)
	if err == nil {
		t.Fatal("expected err but did not get one")
	}
	expectErr := "cannot provide both a reference and a dataset to render"
	if err.Error() != expectErr {
		t.Errorf("error mismatch, expect: %s, got: %s", expectErr, err)
	}
}
