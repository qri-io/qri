package lib

import (
	"testing"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestHistoryRequestsLog(t *testing.T) {
	mr, err := testrepo.NewTestRepo(nil)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	ref, err := mr.GetRef(repo.DatasetRef{Peername: "peer", Name: "movies"})
	if err != nil {
		t.Fatalf("error getting path: %s", err.Error())
	}

	cfg := config.DefaultP2PForTesting()
	cfg.Enabled = false
	node, err := p2p.NewTestableQriNode(mr, cfg)
	if err != nil {
		t.Fatal(err.Error())
	}

	cases := []struct {
		p   *LogParams
		res []repo.DatasetRef
		err string
	}{
		{&LogParams{}, nil, "repo: empty dataset reference"},
		{&LogParams{Ref: repo.DatasetRef{Path: "/badpath"}}, nil, "node is not online and no registry is configured"},
		{&LogParams{Ref: ref}, []repo.DatasetRef{ref}, ""},
	}

	inst := newTestInstanceFromQriNode(node.(*p2p.QriNode))
	req := NewLogMethods(inst)
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
