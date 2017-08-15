package p2p

import (
	"crypto/rand"
	"fmt"

	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
)

// NodeCfg is all configuration options for a Qri Node
type NodeCfg struct {
	PeerId peer.ID // peer identifier

	PubKey  crypto.PubKey
	PrivKey crypto.PrivKey

	// default port to bind tcp listener to
	// ignored if Addrs is supplied
	Port int
	// List of multiaddresses to listen on
	Addrs []ma.Multiaddr
	// secure connection flag. if true
	// PubKey & PrivKey are required
	Secure bool
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
		PeerId:  pid,
		PrivKey: priv,
		PubKey:  pub,

		// Addrs:  []ma.Multiaddr{addr},
		Secure: true,
	}
}

// Validate confirms that the given settings will work, returning an error if not.
func (cfg *NodeCfg) Validate() error {

	// If no listening addresses are set, allocate
	// a tcp multiaddress on local host bound to the default port
	if cfg.Addrs == nil {
		// find an open tcp port
		port, err := LocalOpenPort("tcp", cfg.Port)
		if err != nil {
			return err
		}

		// Create a multiaddress
		addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
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
