package p2p

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/qri-io/analytics"
	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	"gx/ipfs/QmWRCn8vruNAzHx8i6SAXinuheRitKEGu8c7m26stKvsYx/go-testutil"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	// swarm "gx/ipfs/QmdQFrFnPrKRQtpeHKjZ3cVNwxmGKKS2TvhJTuN9C9yduh/go-libp2p-swarm"
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
		Peername: fmt.Sprintf("tes-repo-%d", repoID),
	}, cafs.NewMapstore(), repo.MemPeers{}, &analytics.Memstore{})
}

func NewTestNetwork(ctx context.Context, t *testing.T, num int) ([]*QriNode, error) {
	nodes := make([]*QriNode, 0, num)

	for i := 0; i < num; i++ {
		localnp := testutil.RandPeerNetParamsOrFatal(t)

		r, err := NewTestRepo()
		if err != nil {
			return nil, fmt.Errorf("error creating test repo: %s", err.Error())
		}

		node, err := NewQriNode(r, func(o *NodeCfg) {
			// o.Port = localnp.Addr.Protocols()
			o.QriBootstrapAddrs = []string{}
			o.Addrs = []ma.Multiaddr{
				localnp.Addr,
			}
			o.PrivKey = localnp.PrivKey
			o.PeerID = localnp.ID
		})
		if err != nil {
			return nil, fmt.Errorf("error creating test node: %s", err.Error())
		}
		node.QriPeers.AddPubKey(localnp.ID, localnp.PubKey)
		node.QriPeers.AddPrivKey(localnp.ID, localnp.PrivKey)

		nodes = append(nodes, node)
	}
	return nodes, nil
}

func connectNodes(ctx context.Context, t *testing.T, nodes []*QriNode) {
	var wg sync.WaitGroup
	connect := func(n *QriNode, dst peer.ID, addr ma.Multiaddr) {
		t.Logf("dialing %s from %s\n", n.Identity, dst)
		n.QriPeers.AddAddr(dst, addr, pstore.PermanentAddrTTL)
		// if sw, ok := n.Host.Network().(*swarm.Swarm); ok {
		// 	if _, err := sw.Dial(ctx, dst); err != nil {
		// 	}
		if _, err := n.Host.Network().DialPeer(ctx, dst); err != nil {
			t.Fatal("error swarm dialing to peer", err)
		}
		// }
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

	for _, n := range nodes {
		log.Infof("%s swarm routing table: %s\n", n.Identity, n.Peers())
	}
}

// func connectSwarms2(t *testing.T, ctx context.Context, swarms []*swarm.Swarm) {

// 	var wg sync.WaitGroup
// 	connect := func(s *swarm.Swarm, dst peer.ID, addr ma.Multiaddr) {
// 		// TODO: make a DialAddr func.
// 		s.peers.AddAddr(dst, addr, pstore.PermanentAddrTTL)
// 		if _, err := s.Dial(ctx, dst); err != nil {
// 			t.Fatal("error swarm dialing to peer", err)
// 		}
// 		wg.Done()
// 	}

// 	log.Info("Connecting swarms simultaneously.")
// 	for i, s1 := range swarms {
// 		for _, s2 := range swarms[i+1:] {
// 			wg.Add(1)
// 			connect(s1, s2.LocalPeer(), s2.ListenAddresses()[0]) // try the first.
// 		}
// 	}
// 	wg.Wait()

// 	for _, s := range swarms {
// 		log.Infof("%s swarm routing table: %s", s.local, s.Peers())
// 	}
// }
