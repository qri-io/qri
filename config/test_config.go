package config

import (
	"github.com/qri-io/qri/config/test"
)

// DefaultConfigForTesting constructs a config with precomputed keys, only used for testing.
func DefaultConfigForTesting() *Config {
	info := test.GetTestPeerInfo(0)
	cfg := DefaultConfig()
	cfg.P2P.PrivKey = info.EncodedPrivKey
	cfg.P2P.PeerID = info.EncodedPeerID
	cfg.Profile.Peername = "default_profile_for_testing"
	cfg.Profile.PrivKey = info.EncodedPrivKey
	cfg.Profile.ID = info.EncodedPeerID
	cfg.CLI.ColorizeOutput = false
	return cfg
}

// DefaultProfileForTesting constructs a profile with precomputed keys, only used for testing.
func DefaultProfileForTesting() *ProfilePod {
	info := test.GetTestPeerInfo(0)
	pro := DefaultProfile()
	pro.Peername = "default_profile_for_testing"
	pro.PrivKey = info.EncodedPrivKey
	pro.ID = info.EncodedPeerID
	return pro
}

// DefaultP2PForTesting constructs a p2p with precomputed keys, only used for testing.
func DefaultP2PForTesting() *P2P {
	info := test.GetTestPeerInfo(0)
	p := DefaultP2P()
	p.BootstrapAddrs = nil
	p.PrivKey = info.EncodedPrivKey
	p.PeerID = info.EncodedPeerID
	return p
}
