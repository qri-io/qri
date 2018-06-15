package lib

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/rpc"
	"testing"

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
	mockGateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tmpl := `<html><h1>{{.Peername}}/{{.Name}}</h1></html>`
		w.Write([]byte(tmpl))
	}))

	Config.P2P.HTTPGatewayAddr = mockGateway.URL

	cases := []struct {
		params *RenderParams
		expect []byte
		err    string
	}{
		{&RenderParams{}, nil, "cannot parse empty string as dataset reference"},
		{&RenderParams{Ref: "foo/invalid_ref"}, nil, "error fetching peer from store: profile: not found"},
		{&RenderParams{Ref: "me/movies", Template: []byte("{{ .Meta.Title }}")}, []byte("example movie data"), ""},
		{&RenderParams{Ref: "me/movies", Template: []byte("{{ .BadTemplateBooPlzFail }}")}, nil, `template: template:1:3: executing "template" at <.BadTemplateBooPlzFa...>: can't evaluate field BadTemplateBooPlzFail in type *dataset.DatasetPod`},
		{&RenderParams{Ref: "me/movies", Template: []byte("{{ .BadTemplateBooPlzFail")}, nil, `parsing template: template: template:1: unclosed action`},
		{&RenderParams{Ref: "me/movies"}, []byte("<html><h1>peer/movies</h1></html>"), ""},
		{&RenderParams{Ref: "me/sitemap"}, []byte("<html><h1>peer/sitemap</h1></html>"), ""},
		{&RenderParams{Ref: "me/sitemap", Limit: 4, Offset: 4}, []byte("<html><h1>peer/sitemap</h1></html>"), ""},
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
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			return
		}

		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(string(got), string(c.expect), false)
		if len(diffs) > 1 {
			t.Log(dmp.DiffPrettyText(diffs))
			t.Errorf("case %d failed to match.", i)
		}
	}
}
