package remote

import (
	"testing"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/config"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func TestAddDataset(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	var psClient *PeerSyncClient
	var nilClient Client
	nilClient = psClient
	if err := nilClient.AddDataset(tr.Ctx, &repo.DatasetRef{}, ""); err != ErrNoRemoteClient {
		t.Errorf("nil add mismatch. expected: '%s', got: '%s'", ErrNoRemoteClient, err)
	}

	worldBankRef := writeWorldBankPopulation(tr.Ctx, t, tr.NodeA.Repo)

	cli, err := NewClient(tr.NodeB)
	if err != nil {
		t.Fatal(err)
	}

	tr.NodeA.GoOnline()
	tr.NodeB.GoOnline()

	if err := cli.AddDataset(tr.Ctx, &repo.DatasetRef{Peername: "foo", Name: "bar"}, ""); err == nil {
		t.Error("expected add of invalid ref to error")
	}

	if err := cli.AddDataset(tr.Ctx, &worldBankRef, ""); err != nil {
		t.Error(err.Error())
	}
}

func newMemRepoTestNode(t *testing.T) *p2p.QriNode {
	ms := cafs.NewMapstore()
	pi := cfgtest.GetTestPeerInfo(0)
	pro := &profile.Profile{
		Peername: "remote_test_peer",
		ID:       profile.ID(pi.PeerID),
		PrivKey:  pi.PrivKey,
	}
	mr, err := repo.NewMemRepo(pro, ms, newTestFS(ms), profile.NewMemStore())
	if err != nil {
		t.Fatal(err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}
	return node
}

func newTestFS(cafsys cafs.Filestore) qfs.Filesystem {
	return qfs.NewMux(map[string]qfs.Filesystem{
		"cafs": cafsys,
	})
}

// Convert from test nodes to non-test nodes.
// copied from p2p/peers_test.go
func asQriNodes(testPeers []p2ptest.TestablePeerNode) []*p2p.QriNode {
	// Convert from test nodes to non-test nodes.
	peers := make([]*p2p.QriNode, len(testPeers))
	for i, node := range testPeers {
		peers[i] = node.(*p2p.QriNode)
	}
	return peers
}

func connectMapStores(peers []*p2p.QriNode) {
	for i, s0 := range peers {
		for _, s1 := range peers[i+1:] {
			m0 := (s0.Repo.Store()).(*cafs.MapStore)
			m1 := (s1.Repo.Store()).(*cafs.MapStore)
			m0.AddConnection(m1)
		}
	}
}
