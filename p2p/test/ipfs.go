package p2ptest

import (
	"context"
	"encoding/base64"
	"fmt"

	datastore "github.com/ipfs/go-datastore"
	syncds "github.com/ipfs/go-datastore/sync"
	config "github.com/ipfs/go-ipfs-config"
	core "github.com/ipfs/go-ipfs/core"
	corebs "github.com/ipfs/go-ipfs/core/bootstrap"
	coreapi "github.com/ipfs/go-ipfs/core/coreapi"
	mock "github.com/ipfs/go-ipfs/core/mock"
	keystore "github.com/ipfs/go-ipfs/keystore"
	repo "github.com/ipfs/go-ipfs/repo"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	peer "github.com/libp2p/go-libp2p-core/peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	qfs "github.com/qri-io/qfs"
	ipfsfs "github.com/qri-io/qfs/cafs/ipfs"
	cfgtest "github.com/qri-io/qri/config/test"
	qrirepo "github.com/qri-io/qri/repo"
	profile "github.com/qri-io/qri/repo/profile"
)

// MakeRepoFromIPFSNode wraps an ipfs node with a mock qri repo
func MakeRepoFromIPFSNode(node *core.IpfsNode, username string) (qrirepo.Repo, error) {
	p := &profile.Profile{
		ID:       profile.IDFromPeerID(node.Identity),
		Peername: username,
		PrivKey:  node.PrivateKey,
	}

	store, err := ipfsfs.NewFilestore(func(cfg *ipfsfs.StoreCfg) {
		cfg.Node = node
	})
	if err != nil {
		return nil, err
	}

	fsys := qfs.NewMux(map[string]qfs.Filesystem{
		"cafs": qfs.NewMemFS(),
		"ipfs": store,
	})

	return qrirepo.NewMemRepo(p, store, fsys, profile.NewMemStore())
}

// MakeIPFSNode creates a single mock IPFS Node
func MakeIPFSNode(ctx context.Context) (*core.IpfsNode, coreiface.CoreAPI, error) {
	nd, api, err := MakeIPFSSwarm(ctx, true, 1)
	if err != nil {
		return nil, nil, err
	}

	return nd[0], api[0], nil
}

const testPeerID = "QmTFauExutTsy4XP6JbMFcw2Wa9645HJt2bTqL6qYDCKfe"

// MakeIPFSSwarm creates and connects n number of mock IPFS Nodes
func MakeIPFSSwarm(ctx context.Context, fullIdentity bool, n int) ([]*core.IpfsNode, []coreiface.CoreAPI, error) {
	if n > 10 {
		return nil, nil, fmt.Errorf("cannot generate a network of more than 10 peers")
	}
	mn := mocknet.New(ctx)

	nodes := make([]*core.IpfsNode, n)
	apis := make([]coreiface.CoreAPI, n)

	for i := 0; i < n; i++ {
		var ident config.Identity
		if fullIdentity {

			pi := cfgtest.GetTestPeerInfo(i)
			sk, pk := pi.PrivKey, pi.PubKey

			id, err := peer.IDFromPublicKey(pk)
			if err != nil {
				return nil, nil, err
			}

			kbytes, err := sk.Bytes()
			if err != nil {
				return nil, nil, err
			}

			ident = config.Identity{
				PeerID:  id.Pretty(),
				PrivKey: base64.StdEncoding.EncodeToString(kbytes),
			}
		} else {
			ident = config.Identity{
				PeerID: testPeerID,
			}
		}

		c := config.Config{}
		c.Addresses.Swarm = []string{fmt.Sprintf("/ip4/127.0.%d.1/tcp/4001", i)}
		c.Identity = ident

		r := &repo.Mock{
			C: c,
			D: syncds.MutexWrap(datastore.NewMapDatastore()),
			K: keystore.NewMemKeystore(),
		}

		node, err := core.NewNode(ctx, &core.BuildCfg{
			Repo:   r,
			Host:   mock.MockHostOption(mn),
			Online: fullIdentity,
			ExtraOpts: map[string]bool{
				"pubsub": true,
			},
		})
		if err != nil {
			return nil, nil, err
		}
		nodes[i] = node
		apis[i], err = coreapi.NewCoreAPI(node)
		if err != nil {
			return nil, nil, err
		}
	}

	err := mn.LinkAll()
	if err != nil {
		return nil, nil, err
	}

	bsinf := corebs.BootstrapConfigWithPeers(
		[]pstore.PeerInfo{
			nodes[0].Peerstore.PeerInfo(nodes[0].Identity),
		},
	)

	for _, n := range nodes[1:] {
		if err := n.Bootstrap(bsinf); err != nil {
			return nil, nil, err
		}
	}

	return nodes, apis, nil
}
