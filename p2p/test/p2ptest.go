package p2ptest

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/test"
	testutil "gx/ipfs/QmVvkK7s5imCiq3JVbL3pGfnhcCnf3LrFJPF4GE2sAoGZf/go-testutil"
	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	pstore "gx/ipfs/QmXauCuJzmzapetmC6W4TuDJLL1yFFrVzSHoWv8YdbmnxH/go-libp2p-peerstore"
	"testing"
)

// TestablePeerNode is used by tests only. Implemented by QriNode, whose Keys is QriPeers.
type TestablePeerNode interface {
	Keys() pstore.KeyBook
}

// NodeMakerFunc is a function that constructs a Node from a Repo and options.
type NodeMakerFunc func(repo.Repo, ...func(*config.P2P)) (TestablePeerNode, error)

// NewTestNetwork constructs nodes to test p2p functionality.
func NewTestNetwork(ctx context.Context, t *testing.T, num int, maker NodeMakerFunc) ([]TestablePeerNode, error) {
	nodes := make([]TestablePeerNode, num)

	for i := 0; i < num; i++ {
		rid, err := testutil.RandPeerID()
		if err != nil {
			return nil, fmt.Errorf("error creating peer ID: %s", err.Error())
		}

		r, err := test.NewTestRepoFromProfileID(profile.ID(rid))
		if err != nil {
			return nil, fmt.Errorf("error creating test repo: %s", err.Error())
		}

		node, err := NewTestQriNode(r, t, maker)
		if err != nil {
			return nil, err
		}

		nodes[i] = node
	}
	return nodes, nil
}

// NewTestQriNode constructs a node for testing.
func NewTestQriNode(r repo.Repo, t *testing.T, maker NodeMakerFunc) (TestablePeerNode, error) {
	localnp := testutil.RandPeerNetParamsOrFatal(t)
	data, err := localnp.PrivKey.Bytes()
	if err != nil {
		return nil, err
	}

	privKey := base64.StdEncoding.EncodeToString(data)

	node, err := maker(r, func(c *config.P2P) {
		c.PeerID = localnp.ID.Pretty()
		c.PrivKey = privKey
		c.Addrs = []ma.Multiaddr{
			localnp.Addr,
		}
		c.QriBootstrapAddrs = []string{}
	})
	if err != nil {
		return nil, fmt.Errorf("error creating test node: %s", err.Error())
	}
	node.Keys().AddPubKey(localnp.ID, localnp.PubKey)
	node.Keys().AddPrivKey(localnp.ID, localnp.PrivKey)

	return node, err
}
