package p2p

import (
	"context"
	"fmt"
	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"

	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	swarm "github.com/libp2p/go-libp2p-swarm"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	ma "github.com/multiformats/go-multiaddr"
	msmux "github.com/whyrusleeping/go-smux-multistream"
	yamux "github.com/whyrusleeping/go-smux-yamux"
)

// QriNode encapsulates a qri peer-to-peer node
type QriNode struct {
	Identity   peer.ID        // the local node's identity
	privateKey crypto.PrivKey // the local node's private Key

	Online    bool      // is this node online?
	Host      host.Host // p2p Host
	Discovery discovery.Service
	Peerstore pstore.Peerstore // storage for other Peer instances

	Repo  repo.Repo
	Store cafs.Filestore
}

// NewQriNode creates a new node, providing no arguments will use
// default configuration
func NewQriNode(store cafs.Filestore, options ...func(o *NodeCfg)) (*QriNode, error) {
	if store == nil {
		return nil, fmt.Errorf("need a store to create a qri node")
	}

	cfg := DefaultNodeCfg()
	for _, opt := range options {
		opt(cfg)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// fmt.Println(cfg.Addrs)

	// Create a peerstore
	ps := pstore.NewPeerstore()

	host, err := makeBasicHost(ps, cfg)
	if err != nil {
		return nil, err
	}

	node := &QriNode{
		Identity:  cfg.PeerId,
		Host:      host,
		Online:    cfg.Online,
		Peerstore: ps,
		Repo:      cfg.Repo,
		Store:     store,
	}

	host.SetStreamHandler(QriProtocolId, node.MessageStreamHandler)

	if cfg.Online {
		if err = node.StartDiscovery(); err != nil {
			return nil, err
		}
	}

	return node, nil
}

// Repo gives this node's repository
// func (n *QriNode) Repo() repo.Repo {
// 	return n.repo
// }

// Encapsulated Addresses returns a slice of full multaddrs for this node
func (qn *QriNode) EncapsulatedAddresses() []ma.Multiaddr {
	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", qn.Host.ID().Pretty()))

	res := make([]ma.Multiaddr, len(qn.Host.Addrs()))
	for i, a := range qn.Host.Addrs() {
		res[i] = a.Encapsulate(hostAddr)
	}

	return res
}

// PeerInfo gives an overview of information about this Peer, used in handshaking
// with other peers
// func (n *QriNode) PeerInfo() (map[string]interface{}, error) {
// 	repo.QueryPeers(n.Repo.Peers(), query.Query{

// 		})
// 	if err != nil {
// 		return nil, err
// 	}

// 	return map[string]interface{}{
// 		"Id":        n.Identity.String(),
// 		"namespace": ns,
// 	}, nil
// }

// makeBasicHost creates a LibP2P host from a NodeCfg
func makeBasicHost(ps pstore.Peerstore, cfg *NodeCfg) (host.Host, error) {
	// If using secio, we add the keys to the peerstore
	// for this peer ID.
	if cfg.Secure {
		ps.AddPrivKey(cfg.PeerId, cfg.PrivKey)
		ps.AddPubKey(cfg.PeerId, cfg.PubKey)
	}

	// Set up stream multiplexer
	tpt := msmux.NewBlankTransport()
	tpt.AddTransport("/yamux/1.0.0", yamux.DefaultTransport)

	// Create swarm (implements libP2P Network)
	swrm, err := swarm.NewSwarmWithProtector(
		context.Background(),
		cfg.Addrs,
		cfg.PeerId,
		ps,
		nil,
		tpt,
		nil,
	)
	if err != nil {
		return nil, err
	}

	netw := (*swarm.Network)(swrm)
	basicHost := bhost.New(netw)
	return basicHost, nil
}
