package core

import (
	"testing"

	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo/profile"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestPeerRequestsList(t *testing.T) {
	cases := []struct {
		p   *ListParams
		res []*profile.Profile
		err string
	}{
		{&ListParams{}, nil, ""},
		// {&ListParams{Data: badDataFile}, nil, "error determining dataset schema: no file extension provided"},
		// {&ListParams{DataFilename: badDataFile.FileName(), Data: badDataFile}, nil, "error determining dataset schema: EOF"},
		// {&ListParams{DataFilename: jobsByAutomationFile.FileName(), Data: jobsByAutomationFile}, nil, ""},
		// TODO - need a test that confirms that this node's identity is never present in peers list
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	// TODO - need to upgrade this to include a mock node
	req := NewPeerRequests(&p2p.QriNode{Repo: mr}, nil)
	for i, c := range cases {
		got := []*profile.Profile{}
		err := req.List(c.p, &got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}
