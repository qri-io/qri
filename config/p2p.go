package config

import (
	"encoding/base64"
	"fmt"
	"reflect"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/qri-io/jsonschema"
)

// P2P encapsulates configuration options for qri peer-2-peer communication
type P2P struct {
	// Enabled is a flag for weather this node should connect
	// to the distributed network
	Enabled bool `json:"enabled"`

	// PeerID is this nodes peer identifier
	PeerID string `json:"peerid"`

	PrivKey string `json:"privkey"`

	// Port default port to bind a tcp listener to
	// ignored if Addrs is supplied
	Port int `json:"port"`

	// List of multiaddresses to listen on
	Addrs []ma.Multiaddr `json:"addrs"`

	// QriBootstrapAddrs lists addresses to bootstrap qri node from
	QriBootstrapAddrs []string `json:"qribootstrapaddrs"`

	// list of addresses to bootsrap qri peers on
	BootstrapAddrs []string `json:"bootstrapaddrs"`

	// Enable AutoNAT service. unless you're hosting a server, leave this as false
	AutoNAT bool `json:"autoNAT"`
}

// SetArbitrary is an interface implementation of base/fill/struct in order to safely
// consume config files that have definitions beyond those specified in the struct.
// This simply ignores all additional fields at read time.
func (cfg *P2P) SetArbitrary(key string, val interface{}) error {
	return nil
}

// DefaultP2P generates a p2p struct with only bootstrap addresses set
func DefaultP2P() *P2P {
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
			"/ip4/35.239.80.82/tcp/4001/ipfs/QmdpGkbqDYRPCcwLYnEm8oYGz2G9aUZn9WwPjqvqw3XUAc",   // red
			"/ip4/35.225.152.38/tcp/4001/ipfs/QmTRqTLbKndFC2rp6VzpyApxHCLrFV35setF1DQZaRWPVf",  // orange
			"/ip4/35.202.155.225/tcp/4001/ipfs/QmegNYmwHUQFc3v3eemsYUVf3WiSg4RcMrh3hovA5LncJ2", // yellow
			"/ip4/35.238.10.180/tcp/4001/ipfs/QmessbA6uGLJ7HTwbUJ2niE49WbdPfzi27tdYXdAaGRB4G",  // green
			"/ip4/35.238.105.35/tcp/4001/ipfs/Qmc353gHY5Wx5iHKHPYj3QDqHP4hVA1MpoSsT6hwSyVx3r",  // blue
			"/ip4/35.239.138.186/tcp/4001/ipfs/QmT9YHJF2YkysLqWhhiVTL5526VFtavic3bVueF9rCsjVi", // indigo
			"/ip4/35.226.44.58/tcp/4001/ipfs/QmQS2ryqZrjJtPKDy9VTkdPwdUSpTi1TdpGUaqAVwfxcNh",   // violet
		},
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
	if string(cfg.PeerID) == "" {
		return peer.ID(""), fmt.Errorf("empty string for peer ID")
	}
	return peer.IDB58Decode(cfg.PeerID)
}

// Validate validates all fields of p2p returning all errors found.
func (cfg P2P) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "P2P",
    "description": "Config for the p2p",
    "type": "object",
    "required": ["enabled", "peerid", "privkey", "port", "addrs", "qribootstrapaddrs"],
    "properties": {
      "enabled": {
        "description": "When true, peer to peer communication is allowed",
        "type": "boolean"
      },
      "peerid": {
        "description": "The peerid is this nodes peer identifier",
        "type": "string"
      },
      "privkey": {
        "description": "",
        "type": "string"
      },
      "port": {
        "description": "Port to bind a tcp lister to. Field is ignored if addrs is supplied",
        "type": "integer"
      },
      "addrs": {
        "description": "List of multiaddresses to listen on",
        "anyOf": [
          {"type": "array"},
          {"type": "null"}
        ],
        "items": {
          "type": "string"
        }
      },
      "qribootstrapaddrs": {
        "description": "List of addresses to bootstrap the qri node from",
        "type": "array",
        "items": {
          "type": "string"
        }
      },
      "bootstrapaddrs": {
        "description": "List of addresses to bootstrap qri peers on",
        "anyOf": [
          {"type": "array"},
          {"type": "null"}
        ],
        "items": {
          "type": "string"
        }
      }
    }
  }`)
	return validate(schema, &cfg)
}

// Copy returns a deep copy of a p2p struct
func (cfg *P2P) Copy() *P2P {
	res := &P2P{
		Enabled: cfg.Enabled,
		PeerID:  cfg.PeerID,
		PrivKey: cfg.PrivKey,
		Port:    cfg.Port,
	}

	if cfg.QriBootstrapAddrs != nil {
		res.QriBootstrapAddrs = make([]string, len(cfg.QriBootstrapAddrs))
		reflect.Copy(reflect.ValueOf(res.QriBootstrapAddrs), reflect.ValueOf(cfg.QriBootstrapAddrs))
	}

	if cfg.BootstrapAddrs != nil {
		res.BootstrapAddrs = make([]string, len(cfg.BootstrapAddrs))
		reflect.Copy(reflect.ValueOf(res.BootstrapAddrs), reflect.ValueOf(cfg.BootstrapAddrs))
	}

	return res
}
