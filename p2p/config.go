package p2p

import (
	"crypto/rand"
	"fmt"

	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/qri/repo"

	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	crypto "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

// NodeCfg is all configuration options for a Qri Node
type NodeCfg struct {
	// PeerID is this nodes peer identifier
	PeerID  peer.ID
	PubKey  crypto.PubKey
	PrivKey crypto.PrivKey
	// Port default port to bind a tcp listener to
	// ignored if Addrs is supplied
	Port int
	// List of multiaddresses to listen on
	Addrs []ma.Multiaddr
	// QriBootstrapAddrs lists addresses to bootstrap qri node from
	QriBootstrapAddrs []string
	// Secure connection flag. if true
	// PubKey & PrivKey are required. just, like, never set this
	// to false, okay?
	Secure bool
	// Online is a flag for weather this node should connect
	// to the distributed network
	Online bool
}

// DefaultNodeCfg generates sensible settings for a Qri Node
func DefaultNodeCfg() *NodeCfg {
	r := rand.Reader

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, pub, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil
	}

	// Obtain Peer ID from public key
	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return nil
	}

	return &NodeCfg{
		Online:            true,
		PeerID:            pid,
		PrivKey:           priv,
		PubKey:            pub,
		QriBootstrapAddrs: DefaultBootstrapAddresses,
		Secure:            true,
	}
}

// Validate confirms that the given settings will work, returning an error if not.
func (cfg *NodeCfg) Validate(r repo.Repo) error {
	if r == nil {
		return fmt.Errorf("need a qri Repo to create a qri node")
	}
	// if r == nil && cfg.RepoPath != "" {
	// 	repo, err := fs_repo.NewRepo(store, cfg.RepoPath, cfg.canonicalPeerId(store))
	// 	if err != nil {
	// 		return err
	// 	}
	// 	cfg.Repo = repo
	// }

	// If no listening addresses are set, allocate
	// a tcp multiaddress on local host bound to the default port
	if cfg.Addrs == nil {
		// Create a multiaddress
		addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", cfg.Port))
		if err != nil {
			return err
		}
		cfg.Addrs = []ma.Multiaddr{addr}
	}

	if cfg.Secure && cfg.PubKey == nil {
		return fmt.Errorf("NodeCfg error: PubKey is required for Secure communication")
	} else if cfg.Secure && cfg.PrivKey == nil {
		return fmt.Errorf("NodeCfg error: PrivKey is required for Secure communication")
	}

	// TODO - more checks
	return nil
}

// TODO - currently we're in a bit of a debate between using underlying IPFS node
// ids & generated query profile ids, once that's cleared up we can remove this
// method
func (cfg *NodeCfg) canonicalPeerID(store cafs.Filestore) string {
	if ipfsfs, ok := store.(*ipfs_filestore.Filestore); ok {
		return ipfsfs.Node().PeerHost.ID().Pretty()
	}
	return cfg.PeerID.Pretty()
}
