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
	repo "github.com/ipfs/go-ipfs/repo"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	peer "github.com/libp2p/go-libp2p-core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	qfs "github.com/qri-io/qfs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qfs/muxfs"
	qipfs "github.com/qri-io/qfs/qipfs"
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/event"
	profile "github.com/qri-io/qri/profile"
	qrirepo "github.com/qri-io/qri/repo"
)

// MakeRepoFromIPFSNode wraps an ipfs node with a mock qri repo
func MakeRepoFromIPFSNode(ctx context.Context, node *core.IpfsNode, username string, bus event.Bus) (qrirepo.Repo, error) {
	p := &profile.Profile{
		ID:       profile.IDFromPeerID(node.Identity),
		Peername: username,
		PrivKey:  node.PrivateKey,
	}

	// TODO (b5) -  we can't supply the usual {Type: "mem"} {Type: "local"}, configuration options here
	// b/c we want ipfs to be the "DefaultWriteFS", and muxFS doesn't give us an explicit API for setting
	// the write filesystem
	mux, err := muxfs.New(ctx, []qfs.Config{})
	if err != nil {
		return nil, err
	}

	ipfs, err := qipfs.NewFilesystemFromNode(ctx, node)
	if err != nil {
		return nil, err
	}
	if err := mux.SetFilesystem(ipfs); err != nil {
		return nil, err
	}

	localFS, err := localfs.NewFilesystem(ctx, nil)
	if err != nil {
		return nil, err
	}
	if err := mux.SetFilesystem(localFS); err != nil {
		return nil, err
	}

	if err := mux.SetFilesystem(qfs.NewMemFS()); err != nil {
		return nil, err
	}

	return qrirepo.NewMemRepoWithProfile(ctx, p, mux, bus)
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

			kd := testkeys.GetKeyData(i)
			sk, pk := kd.PrivKey, kd.PrivKey.GetPublic()

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
		[]peer.AddrInfo{
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
