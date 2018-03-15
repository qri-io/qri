package p2p

import (
	"context"
	"fmt"
	// "sort"

	"github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"

	yamux "gx/ipfs/QmNWCEvi7bPRcvqAV8AKLGVNoQdArWi7NJayka2SM4XtRe/go-smux-yamux"
	net "gx/ipfs/QmNa31VPzC561NWwRsJLE7nGYZYuuD2QfpK2b1q9BK54J1/go-libp2p-net"
	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	core "gx/ipfs/QmViBzgruNUoLNBnXcx8YWbDNwV8MNGEGKkLo6JGetygdw/go-ipfs/core"
	msmux "gx/ipfs/QmVniQJkdzLZaZwzwMdd3dJTvWiJ1DQEkreVy6hs6h7Vk5/go-smux-multistream"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	crypto "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	host "gx/ipfs/Qmc1XhrFEiSeBNn3mpfg6gEuYCt5im2gYmNVmncsvmpeAk/go-libp2p-host"
	swarm "gx/ipfs/QmdQFrFnPrKRQtpeHKjZ3cVNwxmGKKS2TvhJTuN9C9yduh/go-libp2p-swarm"
	discovery "gx/ipfs/QmefgzMbKZYsmHFkLqxgaTBG9ypeEjrdWRD5WXH4j1cWDL/go-libp2p/p2p/discovery"
	bhost "gx/ipfs/QmefgzMbKZYsmHFkLqxgaTBG9ypeEjrdWRD5WXH4j1cWDL/go-libp2p/p2p/host/basic"
)

// QriNode encapsulates a qri peer-2-peer node
type QriNode struct {
	// ID is the node's identifier both locally & on the network
	// Identity has a relationship to privateKey (hash of PublicKey)
	ID peer.ID
	// private key for encrypted communication & verifying identity
	privateKey crypto.PrivKey

	// Online indicates weather this is node is connected to the p2p network
	Online bool
	// Host for p2p connections. can be provided by an ipfs node
	Host host.Host
	// Discovery service, can be provided by an ipfs node
	Discovery discovery.Service
	// QriPeers is a peerstore of only qri instances
	QriPeers pstore.Peerstore

	// base context for this node
	ctx context.Context

	// Repo is a repository of this node's qri data
	// note that repo's are built upon a cafs.Filestore, which
	// may contain a reference to a functioning IPFS node. In that case
	// QriNode should piggyback non-qri-specific p2p functionality on the
	// ipfs node provided by repo
	Repo repo.Repo

	// BootstrapAddrs is a list of multiaddresses to bootrap *qri* from (not IPFS)
	BootstrapAddrs []string

	// handlers maps this nodes registered handlers. This works in a way similary to a router
	// in traditional client/server models, but messages are flying around all over the place
	// instead of a request/response pattern
	handlers map[MsgType]HandlerFunc
}

// NewQriNode creates a new node, providing no arguments will use
// default configuration
func NewQriNode(r repo.Repo, options ...func(o *NodeCfg)) (node *QriNode, err error) {
	cfg := DefaultNodeCfg()
	for _, opt := range options {
		opt(cfg)
	}
	if err := cfg.Validate(r); err != nil {
		return nil, err
	}

	// hoist store from repo
	store := r.Store()
	// Create a peerstore
	ps := pstore.NewPeerstore()

	node = &QriNode{
		ID:             cfg.PeerID,
		Online:         cfg.Online,
		QriPeers:       ps,
		Repo:           r,
		ctx:            context.Background(),
		BootstrapAddrs: cfg.QriBootstrapAddrs,
	}
	node.handlers = MakeHandlers(node)

	if cfg.Online {
		// If the underlying content-addressed-filestore is an ipfs
		// node, it has built-in p2p, overlay the qri protocol
		// on the ipfs node's p2p connections.
		if ipfsfs, ok := store.(*ipfs_filestore.Filestore); ok {
			ipfsnode := ipfsfs.Node()
			if ipfsnode.PeerHost != nil {
				node.Host = ipfsnode.PeerHost
				// fmt.Println("ipfs host muxer:")
				// ipfsnode.PeerHost.Mux().Ls(os.Stderr)
			}

			if ipfsnode.Discovery != nil {
				node.Discovery = ipfsnode.Discovery
			}
		} else if node.Host == nil {
			node.Host, err = makeBasicHost(node.ctx, ps, cfg)
			if err != nil {
				return nil, err
			}
		}

		// add multistream handler for qri protocol to the host
		// for more info on multistreams check github.com/multformats/go-multistream
		node.Host.SetStreamHandler(QriProtocolID, node.QriStreamHandler)
	}

	return node, nil
}

// StartOnlineServices bootstraps the node to qri & IPFS networks
// and begins NAT discovery
func (n *QriNode) StartOnlineServices(bootstrapped func(string)) error {
	if !n.Online {
		return nil
	}

	bsPeers := make(chan pstore.PeerInfo, len(n.BootstrapAddrs))
	go func() {
		pInfo := <-bsPeers
		bootstrapped(pInfo.ID.Pretty())
	}()

	// need a call here to ensure boostrapped is called at least once
	// TODO - this is an "original node" problem probably solved by being able
	// to start a node with *no* qri peers specified.
	defer bootstrapped("")
	return n.StartDiscovery(bsPeers)
}

// EncapsulatedAddresses returns a slice of full multaddrs for this node
func (n *QriNode) EncapsulatedAddresses() []ma.Multiaddr {
	// Build host multiaddress
	hostAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", n.Host.ID().Pretty()))
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	res := make([]ma.Multiaddr, len(n.Host.Addrs()))
	for i, a := range n.Host.Addrs() {
		res[i] = a.Encapsulate(hostAddr)
	}

	return res
}

// IPFSNode returns the underlying IPFS node if this Qri Node is running on IPFS
func (n *QriNode) IPFSNode() (*core.IpfsNode, error) {
	if ipfsfs, ok := n.Repo.Store().(*ipfs_filestore.Filestore); ok {
		return ipfsfs.Node(), nil
	}
	return nil, fmt.Errorf("not using IPFS")
}

// IPFSPeerID is a shorthand for accessing this node's IPFS Peer ID
func (n *QriNode) IPFSPeerID() (peer.ID, error) {
	node, err := n.IPFSNode()
	if err != nil {
		return "", err
	}
	return node.Identity, nil
}

// IPFSListenAddresses gives the listening addresses of the underlying IPFS node
func (n *QriNode) IPFSListenAddresses() ([]string, error) {
	maddrs := n.EncapsulatedAddresses()
	addrs := make([]string, len(maddrs))
	for i, maddr := range maddrs {
		addrs[i] = maddr.String()
	}
	return addrs, nil
}

// Peers returns a list of currently connected peer IDs
func (n *QriNode) Peers() []peer.ID {
	if n.Host == nil {
		return []peer.ID{}
	}
	conns := n.Host.Network().Conns()
	seen := make(map[peer.ID]struct{})
	peers := make([]peer.ID, 0, len(conns))

	for _, c := range conns {
		p := c.LocalPeer()
		if _, found := seen[p]; found {
			continue
		}

		seen[p] = struct{}{}
		peers = append(peers, p)
	}

	return peers
}

// AddQriPeer negotiates a connection with a peer to get their profile details
// and peer list.
func (n *QriNode) AddQriPeer(pinfo pstore.PeerInfo) error {
	// add this peer to our store
	n.QriPeers.AddAddrs(pinfo.ID, pinfo.Addrs, pstore.TempAddrTTL)

	// if profile, _ := n.Repo.Peers().GetPeer(pinfo.ID); profile != nil {
	// 	// we've already seen this peer
	// 	return nil
	// }

	if _, err := n.RequestProfile(pinfo.ID); err != nil {
		log.Debug(err.Error())
		return err
	}

	// some time later ask for a list of their peers, you know, "for a friend"
	// go func() {
	// time.Sleep(time.Second * 2)
	// n.RequestPeersList(pinfo.ID)
	// }()

	return nil
}

// ConnectedPeers lists all IPFS connected peers
func (n *QriNode) ConnectedPeers() []string {
	if n.Host == nil {
		return []string{}
	}
	conns := n.Host.Network().Conns()
	peers := make([]string, len(conns))
	for i, c := range conns {
		peers[i] = c.RemotePeer().Pretty()
	}

	return peers
}

// ConnectedQriPeers lists all IPFS connected peers that support the
// qri protocol
func (n *QriNode) ConnectedQriPeers() map[peer.ID]*profile.Profile {
	if n.Host == nil {
		return map[peer.ID]*profile.Profile{}
	}
	conns := n.Host.Network().Conns()
	peers := map[peer.ID]*profile.Profile{}
	for _, c := range conns {
		id := c.RemotePeer()
		// if support, err := n.SupportsQriProtocol(id); err == nil && support {
		if p, err := n.Repo.Peers().GetPeer(id); err == nil {
			peers[id] = p
		}
		// }
	}
	return peers
}

// ConnectToPeer takes a raw peer ID & tries to work out a route to that
// peer, explicitly connecting to them.
func (n *QriNode) ConnectToPeer(pid peer.ID) error {
	// first check for local peer info
	if pinfo := n.Host.Peerstore().PeerInfo(pid); pinfo.ID.String() != "" {
		_, err := n.RequestProfile(pinfo.ID)
		return err
	}

	// attempt to use ipfs routing table to discover peer
	ipfsnode, err := n.IPFSNode()
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	pinfo, err := ipfsnode.Routing.FindPeer(context.Background(), pid)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	_, err = n.RequestProfile(pinfo.ID)
	return err
}

// Context returns this node's context
func (n *QriNode) Context() context.Context {
	if n.ctx == nil {
		n.ctx = context.Background()
	}
	return n.ctx
}

// TODO - finish. We need a proper termination & cleanup process
// func (n *QriNode) Close() error {
// 	if node, err := n.IPFSNode(); err == nil {
// 		return node.Close()
// 	}
// }

// makeBasicHost creates a LibP2P host from a NodeCfg
func makeBasicHost(ctx context.Context, ps pstore.Peerstore, cfg *NodeCfg) (host.Host, error) {
	// If using secio, we add the keys to the peerstore
	// for this peer ID.
	if cfg.Secure {
		ps.AddPrivKey(cfg.PeerID, cfg.PrivKey)
		ps.AddPubKey(cfg.PeerID, cfg.PubKey)
	}

	// Set up stream multiplexer
	tpt := msmux.NewBlankTransport()
	tpt.AddTransport("/yamux/1.0.0", yamux.DefaultTransport)

	// Create swarm (implements libP2P Network)
	swrm, err := swarm.NewSwarmWithProtector(
		ctx,
		cfg.Addrs,
		cfg.PeerID,
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

// SendMessage opens a stream & sends a message from p to one ore more peerIDs
func (n *QriNode) SendMessage(msg Message, replies chan Message, pids ...peer.ID) error {
	for _, peerID := range pids {
		if peerID == n.ID {
			// can't send messages to yourself, silly
			continue
		}

		s, err := n.Host.NewStream(n.Context(), peerID, QriProtocolID)
		if err != nil {
			return fmt.Errorf("error opening stream: %s", err.Error())
		}
		defer s.Close()

		ws := WrapStream(s)
		go n.handleStream(ws, replies)
		if err := ws.sendMessage(msg); err != nil {
			return err
		}
	}

	return nil
}

// QriStreamHandler is the handler we register with the multistream muxer
func (n *QriNode) QriStreamHandler(s net.Stream) {
	defer s.Close()
	n.handleStream(WrapStream(s), nil)
}

// handleStream is a for loop which receives and handles messages
// When Message.HangUp is true, it exits. This will close the stream
// on one of the sides. The other side's receiveMessage() will error
// with EOF, thus also breaking out from the loop.
func (n *QriNode) handleStream(ws *WrappedStream, replies chan Message) {
	for {
		// Loop forever, receiving messages until the other end hangs up
		// or something goes wrong
		msg, err := ws.receiveMessage()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Debugf("error receiving message: %s", err.Error())
			break
		}

		if replies != nil {
			go func() { replies <- msg }()
		}

		handler, ok := n.handlers[msg.Type]
		if !ok {
			log.Infof("peer %s sent unrecognized message type '%s', hanging up", n.ID, msg.Type)
			break
		}

		if hangup := handler(ws, msg); hangup {
			break
		}
	}
}

// MakeHandlers generates a map of MsgTypes to their corresponding handler functions
func MakeHandlers(n *QriNode) map[MsgType]HandlerFunc {
	return map[MsgType]HandlerFunc{
		MtPing:        n.handlePing,
		MtProfile:     n.handleProfile,
		MtProfiles:    n.handleProfiles,
		MtDatasetInfo: n.handleDataset,
		MtDatasets:    n.handleDatasetsList,
		// MtSearch:
		// MtPeers:
		// MtNodes:
		// MtDatasetLog:
	}
}
