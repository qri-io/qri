package lib

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/rpc"
	"testing"

	"github.com/qri-io/qri/base"
	testrepo "github.com/qri-io/qri/repo/test"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestNewRenderRequests(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Errorf("expected NewRenderRequests to panic")
		}
	}()

	tr, err := testrepo.NewTestRepo(nil)
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
				Template: []byte("{{ .BadTemplateBooPlzFail }}"),
			}, nil, `template: index.html:1:3: executing "index.html" at <.BadTemplateBooPlzFa...>: can't evaluate field BadTemplateBooPlzFail in type *dataset.Dataset`},
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

	tr, err := testrepo.NewTestRepo(nil)
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	reqs := NewRenderRequests(tr, nil)

	for i, c := range cases {
		got := []byte{}
		err := reqs.Render(c.params, &got)
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
