package logsync

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	cmp "github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/profile"
)

func TestSyncHTTP(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	a, b := tr.DefaultLogsyncs()

	server := httptest.NewServer(HTTPHandler(a))
	defer server.Close()

	ref, err := writeNasdaqLogs(tr.Ctx, tr.A)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := b.NewPull(ref, ""); err == nil {
		t.Errorf("expected invalid remote address to error")
	}

	pull, err := b.NewPull(ref, server.URL)
	if err != nil {
		t.Errorf("creating pull: %s", err)
	}
	pull.Merge = true

	if _, err := pull.Do(tr.Ctx); err != nil {
		t.Fatalf("pulling nasdaq logs %s", err.Error())
	}

	var expect, got []dsref.VersionInfo
	if expect, err = tr.A.Items(tr.Ctx, ref, 0, 100, ""); err != nil {
		t.Error(err)
	}
	if got, err = tr.B.Items(tr.Ctx, ref, 0, 100, ""); err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}

	worldBankRef, err := writeWorldBankLogs(tr.Ctx, tr.B)
	if err != nil {
		t.Fatal(err)
	}

	push, err := b.NewPush(worldBankRef, server.URL)
	if err != nil {
		t.Fatal(err)
	}

	if err = push.Do(tr.Ctx); err != nil {
		t.Error(err)
	}

	if expect, err = tr.B.Items(tr.Ctx, worldBankRef, 0, 100, ""); err != nil {
		t.Error(err)
	}
	if got, err = tr.A.Items(tr.Ctx, worldBankRef, 0, 100, ""); err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}

	if err = b.DoRemove(tr.Ctx, ref, server.URL); err == nil {
		t.Errorf("expected an error removing a log user doesn't own")
	}

	if err = b.DoRemove(tr.Ctx, worldBankRef, server.URL); err != nil {
		t.Errorf("delete err: %s", err)
	}

	if got, err = tr.A.Items(tr.Ctx, worldBankRef, 0, 100, ""); err == nil {
		t.Logf("%v\n", got)
		t.Error("expected an err fetching removed reference")
	}
}

func TestHTTPClientErrors(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()
	authorA := profile.NewAuthorFromProfile(tr.A.Owner())

	c := httpClient{}
	if _, _, err := c.get(tr.Ctx, authorA, dsref.Ref{}); err == nil {
		t.Error("expected error to exist")
	}

	c.URL = "https://not.a.url      .sadfhajksldfjaskl"
	if _, _, err := c.get(tr.Ctx, authorA, dsref.Ref{}); err == nil {
		t.Error("expected error to exist")
	}

	if err := c.put(tr.Ctx, authorA, dsref.Ref{}, nil); err == nil {
		t.Error("expected error to exist")
	}

	if err := c.del(tr.Ctx, authorA, dsref.Ref{}); err == nil {
		t.Error("expected error to exist")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c.URL = server.URL
	if _, _, err := c.get(tr.Ctx, authorA, dsref.Ref{}); err == nil {
		t.Error("expected error to exist")
	}

	if err := c.put(tr.Ctx, authorA, dsref.Ref{}, nil); err == nil {
		t.Error("expected error to exist")
	}

	if err := c.del(tr.Ctx, authorA, dsref.Ref{}); err == nil {
		t.Error("expected error to exist")
	}
}

func TestHTTPHandlerErrors(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()
	handler := HTTPHandler(nil)
	authorA := profile.NewAuthorFromProfile(tr.A.Owner())

	// one test with no author headers:
	r := httptest.NewRequest("GET", "http://remote.qri.io", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()
	if http.StatusBadRequest != resp.StatusCode {
		t.Errorf("no author response code mismatch. expected: %d, got: %d", http.StatusBadRequest, resp.StatusCode)
	}

	cases := []struct {
		description      string
		method, endpoint string
		expectStatus     int
	}{
		{"no author fields", "GET", "", http.StatusBadRequest},
		{"no author fields", "GET", "?ref=foo/bar", http.StatusBadRequest},
		{"no author fields", "PUT", "", http.StatusBadRequest},
		{"no author fields", "DELETE", "", http.StatusBadRequest},
	}

	for _, c := range cases {
		r := httptest.NewRequest(c.method, fmt.Sprintf("http://remote.qri.io%s", c.endpoint), nil)
		addAuthorHTTPHeaders(r.Header, authorA)

		w := httptest.NewRecorder()
		handler(w, r)

		resp := w.Result()
		if c.expectStatus != resp.StatusCode {
			t.Errorf("case '%s', response code mismatch. expected: %d, got: %d", c.description, c.expectStatus, resp.StatusCode)
		}
	}
}
