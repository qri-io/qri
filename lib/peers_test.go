package lib

import (
	"context"
	"strings"
	"testing"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/config"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/repo"
)

func TestPeerMethodsListNoConnection(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	node := newTestQriNode(t)
	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)
	req := NewPeerMethods(inst)
	p := PeerListParams{}
	_, err := req.List(ctx, &p)
	if err == nil {
		t.Errorf("error: req.List should have failed and returned an error")
	} else if !strings.HasPrefix(err.Error(), "error: not connected") {
		t.Errorf("error: unexpected error message: %s", err.Error())
	}
}

func TestPeerMethodsList(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	cases := []struct {
		p   *PeerListParams
		res []*profile.Profile
		err string
	}{
		{&PeerListParams{}, nil, "error: not connected, run `qri connect` in another window"},
		// {&ListParams{Data: badDataFile}, nil, "error determining dataset schema: no file extension provided"},
		// {&ListParams{DataFilename: badDataFile.FileName(), Data: badDataFile}, nil, "error determining dataset schema: EOF"},
		// {&ListParams{DataFilename: jobsByAutomationFile.FileName(), Data: jobsByAutomationFile}, nil, ""},
		// TODO - need a test that confirms that this node's identity is never present in peers list
	}

	node := newTestQriNode(t)
	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)
	m := NewPeerMethods(inst)
	for i, c := range cases {
		_, err := m.List(ctx, c.p)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestConnectedQriProfiles(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	cases := []struct {
		limit     int
		peerCount int
		err       string
	}{
		{100, 0, ""},
	}

	node := newTestQriNode(t)
	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)
	m := NewPeerMethods(inst)
	for i, c := range cases {
		got, err := m.ConnectedQriProfiles(ctx, &c.limit)
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
	ctx, done := context.WithCancel(context.Background())
	defer done()

	cases := []struct {
		limit     int
		peerCount int
		err       string
	}{
		{100, 0, ""},
	}

	node := newTestQriNode(t)
	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)
	m := NewPeerMethods(inst)
	for i, c := range cases {
		got, err := m.ConnectedIPFSPeers(ctx, &c.limit)
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
	ctx, done := context.WithCancel(context.Background())
	defer done()

	cases := []struct {
		p        PeerInfoParams
		refCount int
		err      string
	}{
		{PeerInfoParams{}, 0, "repo: not found"},
		{PeerInfoParams{ProfileID: profile.IDB58MustDecode("QmY1PxkV9t9RoBwtXHfue1Qf6iYob19nL6rDHuXxooAVZa")}, 0, "repo: not found"},
	}

	node := newTestQriNode(t)
	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)
	m := NewPeerMethods(inst)
	for i, c := range cases {
		_, err := m.Info(ctx, &c.p)
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

func TestPeerConnectionsParamsPod(t *testing.T) {
	if p := NewPeerConnectionParamsPod("peername"); p.Peername != "peername" {
		t.Error("expected Peername to be set")
	}

	if p := NewPeerConnectionParamsPod("/ipfs/Foo"); p.NetworkID != "/ipfs/Foo" {
		t.Error("expected NetworkID to be set")
	}

	ma := "/ip4/130.211.198.23/tcp/4001/p2p/QmNX9nSos8sRFvqGTwdEme6LQ8R1eJ8EuFgW32F9jjp2Pb"
	if p := NewPeerConnectionParamsPod(ma); p.Multiaddr != ma {
		t.Errorf("peer Multiaddr mismatch. expected: %q, got: %q", ma, p.Multiaddr)
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, err := repo.NewMemRepoWithProfile(ctx, testPeerProfile, newTestFS(ctx), event.NilBus)
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

func newTestFS(ctx context.Context) *muxfs.Mux {
	mux, err := muxfs.New(ctx, []qfs.Config{
		{Type: "mem"},
		{Type: "local"},
		{Type: "http"},
	})
	if err != nil {
		panic(err)
	}

	return mux
}

func newTestDisconnectedQriNode() (*p2p.QriNode, error) {
	ctx := context.TODO()
	pro := &profile.Profile{PrivKey: privKey}
	r, err := repo.NewMemRepoWithProfile(ctx, pro, newTestFS(ctx), event.NilBus)
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
