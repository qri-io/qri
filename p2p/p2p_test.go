package p2p

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"sync"
	"testing"

	"github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo/test"

	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	pstore "gx/ipfs/QmXauCuJzmzapetmC6W4TuDJLL1yFFrVzSHoWv8YdbmnxH/go-libp2p-peerstore"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

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

			alloc, err := p2ptest.NewTestQriNode(repo, t, NewTestQriNode)
			if err != nil {
				return nil, err
			}
			node, _ := alloc.(*QriNode)
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
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
