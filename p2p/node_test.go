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

var repoId = 0

func NewTestRepo() (repo.Repo, error) {
	repoId++
	ms := memfs.NewMapstore()
	return repo.NewMemRepo(&profile.Profile{
		Username: fmt.Sprintf("tes-repo-%d", repoId),
	}, ms, repo.MemPeers{}, &analytics.Memstore{})
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
	var wg sync.WaitGroup
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
		// wg.Add(1)
		// if err := node.StartOnlineServices(func(string) {
		// 	wg.Done()
		// }); err != nil {
		// 	return nil, fmt.Errorf("error starting online services for node: %s", err.Error())
		// }
		nodes = append(nodes, node)
	}

	wg.Wait()
	// for _, i := range nodes {
	// 	for _, j := range nodes {
	// 		if i != j {
	// 			i.QriPeers.AddAddrs(j.Identity, j.EncapsulatedAddresses(), pstore.ProviderAddrTTL)
	// 		}
	// 	}
	// }

	return nodes, nil
}

func connectNodes(t *testing.T, ctx context.Context, nodes []*QriNode) {
	var wg sync.WaitGroup
	connect := func(n *QriNode, dst peer.ID, addr ma.Multiaddr) {
		// TODO: make a DialAddr func.
		n.QriPeers.AddAddr(dst, addr, pstore.PermanentAddrTTL)
		n.Host.Network().DialPeer(ctx, dst)
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

	for _, node := range nodes {
		log.Infof("%s swarm routing table: %s", node.Host.Network().LocalPeer().Pretty(), node.Host.Network().Peers())
	}
}
