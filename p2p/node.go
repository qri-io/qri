package p2p

import (
	"context"
	"fmt"
	"sort"

	"github.com/ipfs/go-ipfs/core"
	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/qri/repo"

	yamux "gx/ipfs/QmNWCEvi7bPRcvqAV8AKLGVNoQdArWi7NJayka2SM4XtRe/go-smux-yamux"
	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	msmux "gx/ipfs/QmVniQJkdzLZaZwzwMdd3dJTvWiJ1DQEkreVy6hs6h7Vk5/go-smux-multistream"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	crypto "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	host "gx/ipfs/Qmc1XhrFEiSeBNn3mpfg6gEuYCt5im2gYmNVmncsvmpeAk/go-libp2p-host"
	swarm "gx/ipfs/QmdQFrFnPrKRQtpeHKjZ3cVNwxmGKKS2TvhJTuN9C9yduh/go-libp2p-swarm"
	discovery "gx/ipfs/QmefgzMbKZYsmHFkLqxgaTBG9ypeEjrdWRD5WXH4j1cWDL/go-libp2p/p2p/discovery"
	bhost "gx/ipfs/QmefgzMbKZYsmHFkLqxgaTBG9ypeEjrdWRD5WXH4j1cWDL/go-libp2p/p2p/host/basic"
)

// QriNode encapsulates a qri peer-to-peer node
type QriNode struct {
	log        Logger
	Identity   peer.ID        // the local node's identity
	privateKey crypto.PrivKey // the local node's private Key

	Online    bool              // is this node online?
	Host      host.Host         // p2p Host, can be provided by an ipfs node
	Discovery discovery.Service // peer discovery, can be provided by an ipfs node
	QriPeers  pstore.Peerstore  // storage for other qri Peer instances

	Repo  repo.Repo      // repository of this node's qri data
	Store cafs.Filestore // a content addressed filestore for data storage, usually ipfs
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
	if err := cfg.Validate(store); err != nil {
		return nil, err
	}

	// Create a peerstore
	ps := pstore.NewPeerstore()

	node := &QriNode{
		log:      cfg.Logger,
		Identity: cfg.PeerId,
		Online:   cfg.Online,
		QriPeers: ps,
		Repo:     cfg.Repo,
		Store:    store,
	}

	if cfg.Online {
		// If the underlying content-addressed-filestore is an ipfs
		// node, it has built-in p2p, overlay the qri protocol
		// on the ipfs node's p2p connections.
		if ipfsfs, ok := store.(*ipfs_filestore.Filestore); ok {
			// TODO - in this situation we should adopt the keypair
			// if the ipfs node to avoid conflicts.

			ipfsnode := ipfsfs.Node()
			if ipfsnode.PeerHost != nil {
				node.Host = ipfsnode.PeerHost
				// fmt.Println("ipfs host muxer:")
				// ipfsnode.PeerHost.Mux().Ls(os.Stderr)
			}
			if ipfsnode.Discovery != nil {
				node.Discovery = ipfsnode.Discovery
			}
		}

		if node.Host == nil {
			host, err := makeBasicHost(ps, cfg)
			if err != nil {
				return nil, err
			}
			node.Host = host
		}

		// add multistream handler for qri protocol to the host
		// for more info on multistreams check github.com/multformats/go-multistream
		node.Host.SetStreamHandler(QriProtocolId, node.MessageStreamHandler)
	}

	return node, nil
}

func (n *QriNode) StartOnlineServices() error {
	if !n.Online {
		return nil
	}
	return n.StartDiscovery()
}

// Encapsulated Addresses returns a slice of full multaddrs for this node
func (qn *QriNode) EncapsulatedAddresses() []ma.Multiaddr {
	// Build host multiaddress
	hostAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", qn.Host.ID().Pretty()))
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	res := make([]ma.Multiaddr, len(qn.Host.Addrs()))
	for i, a := range qn.Host.Addrs() {
		res[i] = a.Encapsulate(hostAddr)
	}

	return res
}

func (n *QriNode) IpfsNode() (*core.IpfsNode, error) {
	if ipfsfs, ok := n.Store.(*ipfs_filestore.Filestore); ok {
		return ipfsfs.Node(), nil
	}
	return nil, fmt.Errorf("not using IPFS")
}

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

// PrintSwarmAddrs is pulled from ipfs codebase
func PrintSwarmAddrs(node *QriNode) {
	if !node.Online {
		fmt.Println("qri node running in offline mode.")
		return
	}

	var lisAddrs []string
	ifaceAddrs, err := node.Host.Network().InterfaceListenAddresses()
	if err != nil {
		fmt.Printf("failed to read listening addresses: %s\n", err)
	}
	for _, addr := range ifaceAddrs {
		lisAddrs = append(lisAddrs, addr.String())
	}
	sort.Sort(sort.StringSlice(lisAddrs))
	for _, addr := range lisAddrs {
		fmt.Printf("Swarm listening on %s\n", addr)
	}

	var addrs []string
	for _, addr := range node.Host.Addrs() {
		addrs = append(addrs, addr.String())
	}
	sort.Sort(sort.StringSlice(addrs))
	for _, addr := range addrs {
		fmt.Printf("Swarm announcing %s\n", addr)
	}
}
