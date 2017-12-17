package p2p

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/qri-io/analytics"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

func TestNewNode(t *testing.T) {
	r, err := NewTestRepo()
	if err != nil {
		t.Errorf("error creating test repo: %s", err.Error())
		return
	}

	node, err := NewQriNode(r)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
		return
	}
	if node.Online != true {
		t.Errorf("default node online flag should be true")
	}
}

var repoID = 0

func NewTestRepo() (repo.Repo, error) {
	repoID++
	return repo.NewMemRepo(&profile.Profile{
		Username: fmt.Sprintf("tes-repo-%d", repoID),
	}, memfs.NewMapstore(), repo.MemPeers{}, &analytics.Memstore{})
}

func NewTestNetwork() ([]*QriNode, error) {
	cfgs := []struct {
		port int
	}{
		{10000},
		{10001},
		{10002},
	}

	nodes := make([]*QriNode, 0, len(cfgs))
	for _, cfg := range cfgs {
		r, err := NewTestRepo()
		if err != nil {
			return nil, fmt.Errorf("error creating test repo: %s", err.Error())
		}

		node, err := NewQriNode(r, func(o *NodeCfg) {
			o.Port = cfg.port
			o.QriBootstrapAddrs = []string{}
		})
		if err != nil {
			return nil, fmt.Errorf("error creating test node: %s", err.Error())
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func connectNodes(ctx context.Context, t *testing.T, nodes []*QriNode) {
	var wg sync.WaitGroup
	connect := func(n *QriNode, dst peer.ID, addr ma.Multiaddr) {
		n.QriPeers.AddAddr(dst, addr, pstore.PermanentAddrTTL)
		if _, err := n.Host.Network().DialPeer(ctx, dst); err != nil {
			t.Fatal("error swarm dialing to peer", err)
		}
		wg.Done()
	}

	log.Info("Connecting swarms simultaneously.")
	for i, s1 := range nodes {
		for _, s2 := range nodes[i+1:] {
			wg.Add(1)
			connect(s1, s2.Host.Network().LocalPeer(), s2.Host.Network().ListenAddresses()[0]) // try the first.
		}
	}
	wg.Wait()
}
