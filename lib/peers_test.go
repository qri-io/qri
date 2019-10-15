package lib

import (
	"strings"
	"testing"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qfs/httpfs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestPeerRequestsListNoConnection(t *testing.T) {
	req := NewPeerRequests(nil, nil)
	p := PeerListParams{}
	got := []*config.ProfilePod{}
	err := req.List(&p, &got)
	if err == nil {
		t.Errorf("error: req.List should have failed and returned an error")
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

	mr, err := testrepo.NewTestRepo()
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

	node := newTestQriNode(t)
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

	node := newTestQriNode(t)
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

	node := newTestQriNode(t)
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

	node, err := newTestDisconnectedQriNode()
	if err != nil {
		t.Errorf("error creating qri node: %s", err)
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

func newTestQriNode(t *testing.T) *p2p.QriNode {
	ms := cafs.NewMapstore()
	r, err := repo.NewMemRepo(testPeerProfile, ms, newTestFS(ms), profile.NewMemStore())
	if err != nil {
		t.Fatal(err)
	}
	n, err := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode).New(r)
	if err != nil {
		t.Fatal(err)
	}
	node := n.(*p2p.QriNode)
	return node
}

func newTestFS(cafsys cafs.Filestore) qfs.Filesystem {
	return qfs.NewMux(map[string]qfs.Filesystem{
		"local": localfs.NewFS(),
		"http":  httpfs.NewFS(),
		"cafs":  cafsys,
	})
}

func newTestDisconnectedQriNode() (*p2p.QriNode, error) {
	ms := cafs.NewMapstore()
	r, err := repo.NewMemRepo(&profile.Profile{PrivKey: privKey}, ms, newTestFS(ms), profile.NewMemStore())
	if err != nil {
		return nil, err
	}
	p2pconf := config.DefaultP2P()
	// This Node has P2P disabled.
	p2pconf.Enabled = false
	n, err := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode).NewWithConf(r, p2pconf)
	if err != nil {
		return nil, err
	}
	node := n.(*p2p.QriNode)
	return node, err
}
