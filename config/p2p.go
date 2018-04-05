package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	crypto "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

// P2P encapsulates configuration options for qri peer-2-peer communication
type P2P struct {
	// Enabled is a flag for weather this node should connect
	// to the distributed network
	Enabled bool

	// PeerID is this nodes peer identifier
	PeerID string

	PubKey  string
	PrivKey string

	// Port default port to bind a tcp listener to
	// ignored if Addrs is supplied
	Port int

	// List of multiaddresses to listen on
	Addrs []ma.Multiaddr

	// QriBootstrapAddrs lists addresses to bootstrap qri node from
	QriBootstrapAddrs []string

	// ProfileReplication determines what to do when this peer sees messages
	// broadcast by it's own profile (from another peer instance). setting
	// ProfileReplication == "full" will cause this peer to automatically pin
	// any data that is verifyably posted by the same peer
	ProfileReplication string

	// list of addresses to bootsrap qri peers on
	BoostrapAddrs []string
}

// Default generates sensible settings for p2p, generating a new randomized
// private key & peer id
func (P2P) Default() *P2P {
	r := rand.Reader
	p2p := &P2P{
		Enabled: true,
		// DefaultBootstrapAddresses follows the pattern of IPFS boostrapping off known "gateways".
		// This boostrapping is specific to finding qri peers, which are IPFS peers that also
		// support the qri protocol.
		// (we also perform standard IPFS boostrapping when IPFS networking is enabled, and it's almost always enabled).
		// These are addresses to public qri nodes hosted by qri, inc.
		// One day it would be super nice to bootstrap from a stored history & only
		// use these for first-round bootstrapping.
		QriBootstrapAddrs: []string{
			"/ip4/130.211.198.23/tcp/4001/ipfs/QmNX9nSos8sRFvqGTwdEme6LQ8R1eJ8EuFgW32F9jjp2Pb", // mojo
			"/ip4/35.193.162.149/tcp/4001/ipfs/QmTZxETL4YCCzB1yFx4GT1te68henVHD1XPQMkHZ1N22mm", // epa
		},
		ProfileReplication: "full",
	}

	// Generate a key pair for this host
	if priv, pub, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r); err == nil {
		if pdata, err := priv.Bytes(); err == nil {
			p2p.PrivKey = base64.StdEncoding.EncodeToString(pdata)
		}

		// Obtain Peer ID from public key
		if pid, err := peer.IDFromPublicKey(pub); err == nil {
			p2p.PeerID = pid.Pretty()
		}
	}

	return p2p
}

// DecodePrivateKey generates a PrivKey instance from base64-encoded config file bytes
func (cfg *P2P) DecodePrivateKey() (crypto.PrivKey, error) {
	if cfg.PrivKey == "" {
		return nil, fmt.Errorf("missing private key")
	}

	data, err := base64.StdEncoding.DecodeString(cfg.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("decoding private key: %s", err.Error())
	}

	return crypto.UnmarshalPrivateKey(data)
}

// DecodePeerID takes P2P.ID (a string), and decodes it into a peer.ID
func (cfg *P2P) DecodePeerID() (peer.ID, error) {
	return peer.IDB58Decode(cfg.PeerID)
}

// // Validate confirms that the given settings will work, returning an error if not.
// TODO - this validate method is a carry over from a previous incarnation that I'd like to
// ressurrect as a method of conforming overrides to a steady-state while also validating
// the configuration itself. So, for example, p2p.Enabled = false should override the port
// to be an empty string. It might make sense to rename this while we're at it
// func (cfg *NodeCfg) Validate(r repo.Repo) error {
// 	if r == nil {
// 		return fmt.Errorf("need a qri Repo to create a qri node")
// 	}

// 	// If no listening addresses are set, allocate
// 	// a tcp multiaddress on local host bound to the default port
// 	if cfg.Addrs == nil {
// 		// Create a multiaddress
// 		addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", cfg.Port))
// 		if err != nil {
// 			return err
// 		}
// 		cfg.Addrs = []ma.Multiaddr{addr}
// 	}

// 	if cfg.Secure && cfg.PubKey == nil {
// 		return fmt.Errorf("NodeCfg error: PubKey is required for Secure communication")
// 	} else if cfg.Secure && cfg.PrivKey == nil {
// 		return fmt.Errorf("NodeCfg error: PrivKey is required for Secure communication")
// 	}

// 	// TODO - more checks
// 	return nil
// }
