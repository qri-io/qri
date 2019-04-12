package lib

import (
	"strings"
	"testing"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestPeerRequestsListNoConnection(t *testing.T) {
	peerm := NewPeerMethods(nil)
	p := PeerListParams{}
	got := []*config.ProfilePod{}
	err := peerm.List(&p, &got)
	if err == nil {
		t.Errorf("error: peerm.List should have failed and returned an error")
	} else if !strings.HasPrefix(err.Error(), "error: not connected") {
		t.Errorf("error: unexpected error message: %s", err.Error())
	}
}

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
	inst := newTestInstance(t)

	// TODO - need to upgrade this to include a mock node
	peerm := NewPeerMethods(inst)
	for i, c := range cases {
		got := []*config.ProfilePod{}
		err := peerm.List(c.p, &got)

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

	inst := newTestInstance(t)
	peerm := NewPeerMethods(inst)
	for i, c := range cases {
		got := []*config.ProfilePod{}
		err := peerm.ConnectedQriProfiles(&c.limit, &got)
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

	inst := newTestInstance(t)
	peerm := NewPeerRequests(inst)
	for i, c := range cases {
		got := []string{}
		err := peerm.ConnectedIPFSPeers(&c.limit, &got)
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

	inst := newTestInstance(t)
	peerm := NewPeerRequests(inst)
	for i, c := range cases {
		got := config.ProfilePod{}
		err := peerm.Info(&c.p, &got)
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

	node, err := newTestDisconnectedQriNode()
	if err != nil {
		t.Errorf("error creating qri node: %s", err)
		return
	}
	peerm := NewPeerMethods(newTestInstanceFromQriNode(node))
	for i, c := range cases {
		got := []repo.DatasetRef{}
		err := peerm.GetReferences(&c.p, &got)
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

func TestPeerConnectionsParamsPod(t *testing.T) {
	if p := NewPeerConnectionParamsPod("peername"); p.Peername != "peername" {
		t.Error("expected Peername to be set")
	}

	if p := NewPeerConnectionParamsPod("/ipfs/Foo"); p.NetworkID != "/ipfs/Foo" {
		t.Error("expected NetworkID to be set")
	}

	ma := "/ip4/130.211.198.23/tcp/4001/ipfs/QmNX9nSos8sRFvqGTwdEme6LQ8R1eJ8EuFgW32F9jjp2Pb"
	if p := NewPeerConnectionParamsPod(ma); p.Multiaddr != ma {
		t.Error("expected Multiaddr to be set")
	}

	if p := NewPeerConnectionParamsPod("QmNX9nSos8sRFvqGTwdEme6LQ8R1eJ8EuFgW32F9jjp2Pb"); p.ProfileID != "QmNX9nSos8sRFvqGTwdEme6LQ8R1eJ8EuFgW32F9jjp2Pb" {
		t.Error("expected ProfileID to be set")
	}

	p := PeerConnectionParamsPod{NetworkID: "/ipfs/QmNX9nSos8sRFvqGTwdEme6LQ8R1eJ8EuFgW32F9jjp2Pb"}
	if _, err := p.Decode(); err != nil {
		t.Error(err.Error())
	}
	p = PeerConnectionParamsPod{NetworkID: "/ipfs/QmNX"}
	if _, err := p.Decode(); err == nil {
		t.Error("expected invalid decode to error")
	}

	p = PeerConnectionParamsPod{ProfileID: "QmNX9nSos8sRFvqGTwdEme6LQ8R1eJ8EuFgW32F9jjp2Pb"}
	if _, err := p.Decode(); err != nil {
		t.Error(err.Error())
	}
	p = PeerConnectionParamsPod{ProfileID: "21hub2dj23"}
	if _, err := p.Decode(); err == nil {
		t.Error("expected invalid decode to error")
	}

	p = PeerConnectionParamsPod{Multiaddr: "/ip4/130.211.198.23/tcp/4001/ipfs/QmNX9nSos8sRFvqGTwdEme6LQ8R1eJ8EuFgW32F9jjp2Pb"}
	if _, err := p.Decode(); err != nil {
		t.Error(err.Error())
	}
	p = PeerConnectionParamsPod{Multiaddr: "nhuh"}
	if _, err := p.Decode(); err == nil {
		t.Error("expected invalid decode to error")
	}
}
