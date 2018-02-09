package core

import (
	"testing"

	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestSearch(t *testing.T) {
	cases := []struct {
		p   *repo.SearchParams
		res []repo.DatasetRef
		err string
	}{
		{&repo.SearchParams{}, nil, "this repo doesn't support search"},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewSearchRequests(mr, nil)
	for i, c := range cases {
		got := []repo.DatasetRef{}
		err := req.Search(c.p, &got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}

		if len(c.res) != len(got) {
			t.Errorf("case %d log count mismatch. expected: %d, got: %d", len(c.res), len(got))
			continue
		}
	}
}

func TestReindex(t *testing.T) {
	cases := []struct {
		p      *ReindexSearchParams
		expect bool
		err    string
	}{
		{&ReindexSearchParams{}, false, "search reindexing is currently only supported on file-system repos"},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewSearchRequests(mr, nil)
	for i, c := range cases {
		got := false
		err := req.Reindex(c.p, &got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
		if got != c.expect {
			t.Errorf("case %d expected got:", i, c.expect, got)
			continue
		}
	}
}
