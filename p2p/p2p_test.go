package p2p

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"testing"

	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/test"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	testutil "gx/ipfs/QmWRCn8vruNAzHx8i6SAXinuheRitKEGu8c7m26stKvsYx/go-testutil"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

func NewTestNetwork(ctx context.Context, t *testing.T, num int) ([]*QriNode, error) {
	nodes := make([]*QriNode, num)

	for i := 0; i < num; i++ {
		rid, err := testutil.RandPeerID()
		if err != nil {
			return nil, fmt.Errorf("error creating peer ID: %s", err.Error())
		}

		r, err := NewTestRepo(rid)
		if err != nil {
			return nil, fmt.Errorf("error creating test repo: %s", err.Error())
		}

		node, err := newTestQriNode(r, t)
		if err != nil {
			return nil, err
		}

		nodes[i] = node
	}
	return nodes, nil
}

func NewTestDirNetwork(ctx context.Context, t *testing.T) ([]*QriNode, error) {
	dirs, err := ioutil.ReadDir("testdata")
	if err != nil {
		return nil, err
	}

	nodes := []*QriNode{}
	for _, dir := range dirs {
		if dir.IsDir() {
			repo, _, err := test.NewMemRepoFromDir(filepath.Join("testdata", dir.Name()))
			if err != nil {
				return nil, err
			}

			node, err := newTestQriNode(repo, t)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

func newTestQriNode(r repo.Repo, t *testing.T) (*QriNode, error) {
	localnp := testutil.RandPeerNetParamsOrFatal(t)

	node, err := NewQriNode(r, func(o *NodeCfg) {
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

	return node, err
}

func connectNodes(ctx context.Context, nodes []*QriNode) error {
	var wg sync.WaitGroup
	connect := func(n *QriNode, dst peer.ID, addr ma.Multiaddr) error {
		// t.Logf("dialing %s from %s\n", n.ID, dst)
		n.QriPeers.AddAddr(dst, addr, pstore.PermanentAddrTTL)
		// if sw, ok := n.Host.Network().(*swarm.Swarm); ok {
		//  if _, err := sw.Dial(ctx, dst); err != nil {
		//  }
		if _, err := n.Host.Network().DialPeer(ctx, dst); err != nil {
			return err
		}
		// }
		wg.Done()
		return nil
	}

	// log.Info("Connecting swarms simultaneously.")
	for i, s1 := range nodes {
		for _, s2 := range nodes[i+1:] {
			wg.Add(1)
			if err := connect(s1, s2.Host.Network().LocalPeer(), s2.Host.Network().ListenAddresses()[0]); err != nil {
				return err
			}
		}
	}
	wg.Wait()

	// for _, n := range nodes {
	//  // log.Infof("%s swarm routing table: %s\n", n.ID, n.Peers())
	// }
	return nil
}
