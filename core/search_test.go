package core

import (
	"testing"

	// "github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
	"github.com/qri-io/registry/regclient"
	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestSearch(t *testing.T) {
	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	cases := []struct {
		p          *regclient.SearchParams
		numResults int
		err        string
	}{
		{&regclient.SearchParams{"abc", nil, 0, 100}, 0, "error 400: search not supported"},
	}

	req := NewSearchRequests(mr, nil)
	for i, c := range cases {
		got := &[]Result{}
		err := req.Search(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}

		if len(*got) != c.numResults {
			t.Errorf("case %d result count mismatch: expected: %d results, got: %d", i, c.numResults, len(*got))
		}
	}
}

// func TestReindex(t *testing.T) {
// 	cases := []struct {
// 		p      *ReindexSearchParams
// 		expect bool
// 		err    string
// 	}{
// 		{&ReindexSearchParams{}, false, "search reindexing is currently only supported on file-system repos"},
// 	}

// 	mr, err := testrepo.NewTestRepo(nil)
// 	if err != nil {
// 		t.Errorf("error allocating test repo: %s", err.Error())
// 		return
// 	}

// 	req := NewSearchRequests(mr, nil)
// 	for i, c := range cases {
// 		got := false
// 		err := req.Reindex(c.p, &got)

// 		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
// 			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
// 			continue
// 		}
// 		if got != c.expect {
// 			t.Errorf("case %d expected: %t got: %t", i, c.expect, got)
// 			continue
// 		}
// 	}
// }
