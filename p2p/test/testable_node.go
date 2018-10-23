package p2ptest

import (
	"context"
	"fmt"
	"time"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"

	// net "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
	libp2p "gx/ipfs/QmY51bqSM5XgxQZqsBrQcRkKTnCb8EKpJpR9K6Qax7Njco/go-libp2p"
	// ma "gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"
	connmgr "gx/ipfs/QmYAL9JsqVVPFWwM1ZzHNsofmTzRYQHJ2KqQaBmFJjJsNx/go-libp2p-connmgr"
	pstore "gx/ipfs/QmZR2XWVVBCtbgBWnQhWk2xcQfaR3W8faQPriAiaaj7rsr/go-libp2p-peerstore"
	host "gx/ipfs/Qmb8T6YBBsjYsVGfrihQLfCJveczZnneSBqBKkYEBWDjge/go-libp2p-host"
	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
)

// TestableNode satisfies the TestablePeerNode interface
// It is used for testing inside the p2ptest package
type TestableNode struct {
	host host.Host
	cfg  *config.P2P
	Repo repo.Repo
}

// TestProtocolID is the key used to set the stream handler to our
// testing protocol
var TestProtocolID = "qri"

// Host returns the node's underlying host
func (n *TestableNode) Host() host.Host {
	return n.host
}

// SimplePeerInfo returns the PeerInfo of the TestableNode
func (n *TestableNode) SimplePeerInfo() pstore.PeerInfo {
	return pstore.PeerInfo{
		ID:    n.Host().ID(),
		Addrs: n.Host().Addrs(),
	}
}

// UpgradeToQriConnection upgrades the connection from a basic connection
// to a Qri connection
func (n *TestableNode) UpgradeToQriConnection(pstore.PeerInfo) error {
	return nil
}

// GoOnline assumes the TestNode is not online, it will set
// the StreamHandler and updates our profile with the underlying peerIDs
func (n *TestableNode) GoOnline() error {

	// add multistream handler for qri protocol to the host
	// for more info on multistreams check github.com/multformats/go-multistream
	// n.Host().SetStreamHandler(QriProtocolID, n.QriStreamHandler)

	p, err := n.Repo.Profile()
	if err != nil {
		return fmt.Errorf("error getting repo profile: %s", err.Error())
	}
	p.PeerIDs = []peer.ID{n.Host().ID()}

	// update profile with our p2p addresses
	if err := n.Repo.SetProfile(p); err != nil {
		return err
	}

	return nil
}

// NewTestableNode creates a testable node from a repo and a config.P2P
// it creates a basic host
func NewTestableNode(r repo.Repo, p2pconf *config.P2P) (TestablePeerNode, error) {
	ctx := context.Background()
	ps := pstore.NewPeerstore()
	// this is essentially what is located in the p2p.makeBasicHost function
	pk, err := p2pconf.DecodePrivateKey()
	if err != nil {
		return nil, err
	}

	pid, err := p2pconf.DecodePeerID()
	if err != nil {
		return nil, err
	}
	ps.AddPrivKey(pid, pk)
	ps.AddPubKey(pid, pk.GetPublic())
	opts := []libp2p.Option{
		libp2p.ListenAddrs(p2pconf.Addrs...),
		libp2p.Identity(pk),
		libp2p.Peerstore(ps),
		libp2p.ConnectionManager(connmgr.NewConnManager(1000, 0, time.Millisecond)),
	}
	basicHost, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &TestableNode{
		host: basicHost,
		Repo: r,
		cfg:  p2pconf,
	}, nil
}

var _ TestablePeerNode = (*TestableNode)(nil)
var _ NodeMakerFunc = NewTestableNode
