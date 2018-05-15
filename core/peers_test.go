package core

import (
	"testing"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestPeerRequestsList(t *testing.T) {
	cases := []struct {
		p   *PeerListParams
		res []*profile.Profile
		err string
	}{
		{&PeerListParams{}, nil, ""},
		// {&ListParams{Data: badDataFile}, nil, "error determining dataset schema: no file extension provided"},
		// {&ListParams{DataFilename: badDataFile.FileName(), Data: badDataFile}, nil, "error determining dataset schema: EOF"},
		// {&ListParams{DataFilename: jobsByAutomationFile.FileName(), Data: jobsByAutomationFile}, nil, ""},
		// TODO - need a test that confirms that this node's identity is never present in peers list
	}

	mr, err := testrepo.NewTestRepo(nil)
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	// TODO - need to upgrade this to include a mock node
	req := NewPeerRequests(&p2p.QriNode{Repo: mr}, nil)
	for i, c := range cases {
		got := []*config.ProfilePod{}
		err := req.List(c.p, &got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestConnectedQriProfiles(t *testing.T) {
	// TODO - we're going to need network simulation to test this properly
	cases := []struct {
		limit     int
		peerCount int
		err       string
	}{
		{100, 0, ""},
	}

	node, err := testQriNode()
	if err != nil {
		t.Errorf("error creating qri node: %s", err.Error())
		return
	}

	req := NewPeerRequests(node, nil)
	for i, c := range cases {
		got := []*config.ProfilePod{}
		err := req.ConnectedQriProfiles(&c.limit, &got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}
		if len(got) != c.peerCount {
			t.Errorf("case %d peer count mismatch. expected: %d, got: %d", i, c.peerCount, len(got))
			continue
		}
	}
}

func TestConnectedIPFSPeers(t *testing.T) {
	// TODO - we're going to need an IPFS network simulation to test this properly
	cases := []struct {
		limit     int
		peerCount int
		err       string
	}{
		{100, 0, ""},
	}

	node, err := testQriNode()
	if err != nil {
		t.Errorf("error creating qri node: %s", err.Error())
		return
	}

	req := NewPeerRequests(node, nil)
	for i, c := range cases {
		got := []string{}
		err := req.ConnectedIPFSPeers(&c.limit, &got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}
		if len(got) != c.peerCount {
			t.Errorf("case %d peer count mismatch. expected: %d, got: %d", i, c.peerCount, len(got))
			continue
		}
	}
}

func TestInfo(t *testing.T) {
	// TODO - we're going to need an IPFS network simulation to test this properly
	cases := []struct {
		p        PeerInfoParams
		refCount int
		err      string
	}{
		{PeerInfoParams{}, 0, "repo: not found"},
		{PeerInfoParams{ProfileID: profile.IDB58MustDecode("QmY1PxkV9t9RoBwtXHfue1Qf6iYob19nL6rDHuXxooAVZa")}, 0, "repo: not found"},
	}

	node, err := testQriNode()
	if err != nil {
		t.Errorf("error creating qri node: %s", err.Error())
		return
	}

	req := NewPeerRequests(node, nil)
	for i, c := range cases {
		got := config.ProfilePod{}
		err := req.Info(&c.p, &got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}
		// TODO - compare output, first add an Equal method to profile
		// if got. {
		// 	t.Errorf("case %d reference count mismatch. expected: %d, got: %d", i, c.refCount, len(got))
		// 	continue
		// }
	}
}

func TestGetReferences(t *testing.T) {
	// TODO - we're going to need an IPFS network simulation to test this properly
	cases := []struct {
		p        PeerRefsParams
		refCount int
		err      string
	}{
		{PeerRefsParams{}, 0, "error decoding peer Id: input isn't valid multihash"},
		{PeerRefsParams{PeerID: "QmY1PxkV9t9RoBwtXHfue1Qf6iYob19nL6rDHuXxooAVZa"}, 0, "not connected to p2p network"},
	}

	node, err := testQriNode()
	if err != nil {
		t.Errorf("error creating qri node: %s", err.Error())
		return
	}

	req := NewPeerRequests(node, nil)
	for i, c := range cases {
		got := []repo.DatasetRef{}
		err := req.GetReferences(&c.p, &got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}
		if len(got) != c.refCount {
			t.Errorf("case %d reference count mismatch. expected: %d, got: %d", i, c.refCount, len(got))
			continue
		}
	}
}
