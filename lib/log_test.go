package lib

import (
	"testing"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestHistoryRequestsLog(t *testing.T) {
	mr, refs, err := testrepo.NewTestRepoForLog(nil)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
		return
	}

	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	cases := []struct {
		description string
		p           *LogParams
		refs        []repo.DatasetRef
		err         string
	}{
		{"log list - empty",
			&LogParams{}, nil, "repo: empty dataset reference"},
		{"log list - bad path",
			&LogParams{Ref: repo.DatasetRef{Path: "/badpath"}}, nil, "node is not online and no registry is configured"},
		{"log list - default",
			&LogParams{Ref: refs[0]}, refs, ""},
		{"log list - offset 0 limit 3",
			&LogParams{Ref: refs[0], ListParams: ListParams{Offset: 0, Limit: 3}}, refs[:3], ""},
		{"log list - offset 3 limit 3",
			&LogParams{Ref: refs[0], ListParams: ListParams{Offset: 3, Limit: 3}}, refs[3:], ""},
		{"log list - offset 6 limit 3",
			&LogParams{Ref: refs[0], ListParams: ListParams{Offset: 6, Limit: 3}}, []repo.DatasetRef{}, ""},
	}

	req := NewLogRequests(node, nil)
	for _, c := range cases {
		got := []repo.DatasetRef{}
		err := req.Log(c.p, &got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case '%s' error mismatch: expected: %s, got: %s", c.description, c.err, err)
			continue
		}

		if len(c.refs) != len(got) {
			t.Errorf("case '%s' expected log length error. expected: %d, got: %d", c.description, len(c.refs), len(got))
			continue
		}

		for j, expect := range c.refs {
			if err := repo.CompareDatasetRef(expect, got[j]); err != nil {
				t.Errorf("case '%s' expected dataset error. index %d mismatch: %s", c.description, j, err.Error())
				continue
			}
		}
	}
}
