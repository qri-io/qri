package core

import (
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestHistoryRequestsLog(t *testing.T) {
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}
	path, err := mr.GetPath("movies")
	if err != nil {
		t.Errorf("error getting path: %s", err.Error())
		return
	}

	cases := []struct {
		p   *LogParams
		res []*dataset.Dataset
		err string
	}{
		{&LogParams{}, nil, "path is required"},
		{&LogParams{Path: datastore.NewKey("/badpath")}, nil, "error getting file bytes: datastore: key not found"},
		{&LogParams{Path: path}, []*dataset.Dataset{&dataset.Dataset{}}, ""},
	}

	req := NewHistoryRequests(mr)
	for i, c := range cases {
		got := []*dataset.Dataset{}
		err := req.Log(c.p, &got)

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
