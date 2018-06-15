package lib

import (
	"testing"

	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestHistoryRequestsLog(t *testing.T) {
	mr, err := testrepo.NewTestRepo(nil)
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}
	ref, err := mr.GetRef(repo.DatasetRef{Peername: "peer", Name: "movies"})
	if err != nil {
		t.Errorf("error getting path: %s", err.Error())
		return
	}

	cases := []struct {
		p   *LogParams
		res []repo.DatasetRef
		err string
	}{
		{&LogParams{}, nil, "either path or peername/name is required"},
		{&LogParams{Ref: repo.DatasetRef{Path: "/badpath"}}, nil, "error getting reference '@/badpath': repo: not found"},
		{&LogParams{Ref: ref}, []repo.DatasetRef{ref}, ""},
	}

	req := NewHistoryRequests(mr, nil)
	for i, c := range cases {
		got := []repo.DatasetRef{}
		err := req.Log(c.p, &got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}

		if len(c.res) != len(got) {
			t.Errorf("case %d log count mismatch. expected: %d, got: %d", i, len(c.res), len(got))
			continue
		}
	}
}
