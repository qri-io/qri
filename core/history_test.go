package core

import (
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/repo"
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
		res []*repo.DatasetRef
		err string
	}{
		{&LogParams{}, nil, "path is required"},
		{&LogParams{Path: datastore.NewKey("/badpath")}, nil, "error loading dataset: error getting file bytes: datastore: key not found"},
		{&LogParams{Path: path}, []*repo.DatasetRef{&repo.DatasetRef{Path: path}}, ""},
	}

	req := NewHistoryRequests(mr, nil)
	for i, c := range cases {
		got := []*repo.DatasetRef{}
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
