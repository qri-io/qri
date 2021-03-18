package test

import (
	"github.com/qri-io/qfs"
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/config"
)

// DefaultConfigForTesting constructs a config with precomputed keys, only used for testing.
func DefaultConfigForTesting() *config.Config {
	kd := testkeys.GetKeyData(0)
	cfg := config.DefaultConfig()
	cfg.P2P.PrivKey = kd.EncodedPrivKey
	cfg.P2P.PeerID = kd.EncodedPeerID
	cfg.Profile.Peername = "default_profile_for_testing"
	cfg.Profile.PrivKey = kd.EncodedPrivKey
	cfg.Profile.ID = kd.EncodedPeerID
	cfg.CLI.ColorizeOutput = false
	cfg.Registry.Location = ""
	return cfg
}

// DefaultMemConfigForTesting constructs a config with precomputed keys & settings for an in memory repo & fs, only used for testing.
func DefaultMemConfigForTesting() *config.Config {
	cfg := DefaultConfigForTesting().Copy()

	// add settings for in-mem filesystem
	cfg.Filesystems = []qfs.Config{
		{Type: "mem"},
		{Type: "local"},
		{Type: "http"},
	}

	// add settings for in mem repo
	cfg.Repo.Type = "mem"

	return cfg
}

// DefaultProfileForTesting constructs a profile with precompted keys, only used for testing.
// TODO (b5): move this into a new profile/test package
func DefaultProfileForTesting() *config.ProfilePod {
	kd := testkeys.GetKeyData(0)
	pro := config.DefaultProfile()
	pro.Peername = "default_profile_for_testing"
	pro.PrivKey = kd.EncodedPrivKey
	pro.ID = kd.EncodedPeerID
	return pro
}

// DefaultP2PForTesting constructs a p2p with precomputed keys, only used for testing.
func DefaultP2PForTesting() *config.P2P {
	kd := testkeys.GetKeyData(0)
	p := config.DefaultP2P()
	p.BootstrapAddrs = nil
	p.PrivKey = kd.EncodedPrivKey
	p.PeerID = kd.EncodedPeerID
	return p
}
