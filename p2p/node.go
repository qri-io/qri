package p2p

import (
	"context"
	"fmt"
	"sync"

	"github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
	// "github.com/qri-io/qri/repo/profile"

	yamux "gx/ipfs/QmNWCEvi7bPRcvqAV8AKLGVNoQdArWi7NJayka2SM4XtRe/go-smux-yamux"
	discovery "gx/ipfs/QmNh1kGFFdsPu79KNSaL4NUKUPb4Eiz4KHdMtFY6664RDp/go-libp2p/p2p/discovery"
	bhost "gx/ipfs/QmNh1kGFFdsPu79KNSaL4NUKUPb4Eiz4KHdMtFY6664RDp/go-libp2p/p2p/host/basic"
	host "gx/ipfs/QmNmJZL7FQySMtE2BQuLMuZg2EB2CLEunJJUSVSc9YnnbV/go-libp2p-host"
	swarm "gx/ipfs/QmSwZMWwFZSUpe5muU2xgTUwppH24KfMwdPXiwbEp2c6G5/go-libp2p-swarm"
	msmux "gx/ipfs/QmVniQJkdzLZaZwzwMdd3dJTvWiJ1DQEkreVy6hs6h7Vk5/go-smux-multistream"
	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	pstore "gx/ipfs/QmXauCuJzmzapetmC6W4TuDJLL1yFFrVzSHoWv8YdbmnxH/go-libp2p-peerstore"
	net "gx/ipfs/QmXfkENeeBvh3zYA51MaSdGUdBjhQ99cP5WQe8zgr6wchG/go-libp2p-net"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	crypto "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	core "gx/ipfs/QmatUACvrFK3xYg1nd2iLAKfz7Yy5YB56tnzBYHpqiUuhn/go-ipfs/core"
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

	// msgState keeps a "scratch pad" of message IDS & timeouts
	msgState *sync.Map
	// msgChan provides a channel of received messages for others to tune into
	msgChan chan Message
	// receivers is a list of anyone who wants to be notifed on new message arrival
	receivers []chan Message
	// profileReplication sets what to do when this node sees it's own profile
	profileReplication string
}

// Assert that conversions needed by the tests are valid.
var _ p2ptest.TestablePeerNode = (*QriNode)(nil)
var _ p2ptest.NodeMakerFunc = NewTestQriNode

// NewTestQriNode creates a new node, as a TestablePeerNode, usable by testing utilities.
func NewTestQriNode(r repo.Repo, options ...func(o *config.P2P)) (p2ptest.TestablePeerNode, error) {
	return NewQriNode(r, options...)
}

// NewQriNode creates a new node, providing no arguments will use
// default configuration
func NewQriNode(r repo.Repo, options ...func(o *config.P2P)) (node *QriNode, err error) {
	cfg := config.DefaultP2P()
	for _, opt := range options {
		opt(cfg)
	}
	// if err := cfg.Validate(r); err != nil {
	// 	return nil, err
	// }

	// hoist store from repo
	store := r.Store()

	pid, err := cfg.DecodePeerID()
	if err != nil {
		return nil, fmt.Errorf("error decoding peer id: %s", err.Error())
	}

	node = &QriNode{
		ID:                 pid,
		Online:             cfg.Enabled,
		Repo:               r,
		ctx:                context.Background(),
		BootstrapAddrs:     cfg.QriBootstrapAddrs,
		msgState:           &sync.Map{},
		msgChan:            make(chan Message),
		profileReplication: cfg.ProfileReplication,
	}
	node.handlers = MakeHandlers(node)

	if node.Online {
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
			ps := pstore.NewPeerstore()
			node.Host, err = makeBasicHost(node.ctx, ps, cfg)
			if err != nil {
				return nil, fmt.Errorf("error creating host: %s", err.Error())
			}
		}

		// add multistream handler for qri protocol to the host
		// for more info on multistreams check github.com/multformats/go-multistream
		node.Host.SetStreamHandler(QriProtocolID, node.QriStreamHandler)

		p, err := node.Repo.Profile()
		if err != nil {
			log.Errorf("error getting repo profile: %s\n", err.Error())
			return node, err
		}
		p.PeerIDs = []peer.ID{
			node.Host.ID(),
		}
		// add listen addresses to profile store
		// if addrs, err := node.ListenAddresses(); err == nil {
		// 	if p.Addresses == nil {
		// 		p.Addresses = []string{fmt.Sprintf("/ipfs/%s", node.Host.ID().Pretty())}
		// 	}
		// }

		if err := node.Repo.SetProfile(p); err != nil {
			return node, err
		}
	}

	go node.echoMessages()

	return node, nil
}

// StartOnlineServices bootstraps the node to qri & IPFS networks
// and begins NAT discovery
func (n *QriNode) StartOnlineServices(bootstrapped func(string)) error {
	if !n.Online {
		return nil
	}

	bsPeers := make(chan pstore.PeerInfo, len(n.BootstrapAddrs))
	// need a call here to ensure boostrapped is called at least once
	// TODO - this is an "original node" problem probably solved by being able
	// to start a node with *no* qri peers specified.
	defer bootstrapped("")

	go func() {
		pInfo := <-bsPeers
		bootstrapped(pInfo.ID.Pretty())

		if err := n.AnnounceConnected(); err != nil {
			log.Infof("error announcing connected: %s", err.Error())
		}
	}()

	return n.StartDiscovery(bsPeers)
}

// ReceiveMessages adds a listener for newly received messages
func (n *QriNode) ReceiveMessages() chan Message {
	r := make(chan Message)
	n.receivers = append(n.receivers, r)
	return r
}

func (n *QriNode) echoMessages() {
	for {
		msg := <-n.msgChan
		for _, r := range n.receivers {
			r <- msg
		}
	}
}

// IPFSNode returns the underlying IPFS node if this Qri Node is running on IPFS
func (n *QriNode) IPFSNode() (*core.IpfsNode, error) {
	if ipfsfs, ok := n.Repo.Store().(*ipfs_filestore.Filestore); ok {
		return ipfsfs.Node(), nil
	}
	return nil, fmt.Errorf("not using IPFS")
}

// ListenAddresses gives the listening addresses of this node on the p2p network as
// a slice of strings
func (n *QriNode) ListenAddresses() ([]string, error) {
	maddrs := n.EncapsulatedAddresses()
	addrs := make([]string, len(maddrs))
	for i, maddr := range maddrs {
		addrs[i] = maddr.String()
	}
	return addrs, nil
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
func makeBasicHost(ctx context.Context, ps pstore.Peerstore, cfg *config.P2P) (host.Host, error) {
	pk, err := cfg.DecodePrivateKey()
	if err != nil {
		return nil, err
	}

	pid, err := cfg.DecodePeerID()
	if err != nil {
		return nil, err
	}

	ps.AddPrivKey(pid, pk)
	ps.AddPubKey(pid, pk.GetPublic())

	// Set up stream multiplexer
	tpt := msmux.NewBlankTransport()
	tpt.AddTransport("/yamux/1.0.0", yamux.DefaultTransport)

	// Create swarm (implements libP2P Network)
	swrm, err := swarm.NewSwarmWithProtector(
		ctx,
		cfg.Addrs,
		pid,
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
	// defer s.Close()
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
		go func() {
			n.msgChan <- msg
		}()

		handler, ok := n.handlers[msg.Type]
		if !ok {
			log.Infof("peer %s sent unrecognized message type '%s', hanging up", n.ID, msg.Type)
			break
		}

		if hangup := handler(ws, msg); hangup {
			break
		}
	}

	ws.stream.Close()
}

// Keys returns the KeyBook for the node.
func (n *QriNode) Keys() pstore.KeyBook {
	return n.Host.Peerstore()
}

// Addrs returns the AddrBook for the node.
func (n *QriNode) Addrs() pstore.AddrBook {
	return n.Host.Peerstore()
}

// HostNetwork returns the Host's Network for the node.
func (n *QriNode) HostNetwork() net.Network {
	return n.Host.Network()
}

// MakeHandlers generates a map of MsgTypes to their corresponding handler functions
func MakeHandlers(n *QriNode) map[MsgType]HandlerFunc {
	return map[MsgType]HandlerFunc{
		MtPing:        n.handlePing,
		MtProfile:     n.handleProfile,
		MtProfiles:    n.handleProfiles,
		MtDatasetInfo: n.handleDataset,
		MtDatasets:    n.handleDatasetsList,
		MtEvents:      n.handleEvents,
		MtConnected:   n.handleConnected,
		// MtSearch:
		// MtPeers:
		// MtNodes:
		// MtDatasetLog:
	}
}
