// Package migrate defines migrations for qri configuration files
package migrate

import (
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
)

// RunMigrations checks to see if any migrations runs them
func RunMigrations(streams ioes.IOStreams, cfg *config.Config) (migrated bool, err error) {
	if cfg.Revision != config.CurrentConfigRevision {
		streams.PrintErr("migrating configuration...")
		if err := ZeroToOne(cfg); err != nil {
			return false, err
		}
		streams.PrintErr("done!\n")
		return true, nil
	}
	return false, nil
}

// ZeroToOne migrates a configuration from Revision Zero (no revision number) to Revision 1
func ZeroToOne(cfg *config.Config) error {
	if cfg.P2P != nil {
		removes := map[string]bool{
			"/ip4/130.211.198.23/tcp/4001/ipfs/QmNX9nSos8sRFvqGTwdEme6LQ8R1eJ8EuFgW32F9jjp2Pb": true, // mojo
			"/ip4/35.193.162.149/tcp/4001/ipfs/QmTZxETL4YCCzB1yFx4GT1te68henVHD1XPQMkHZ1N22mm": true, // epa
			"/ip4/35.226.92.45/tcp/4001/ipfs/QmP6sbnHXANXgQ7JeCCeCKdJrgpvUd8s75YNfzdkHf6Mpi":   true, // 538
			"/ip4/35.192.140.245/tcp/4001/ipfs/QmUUVNiTz2K9zQSH9PxerKWXmN1p3DBo3oJXurvYziFzqh": true, // EDGI
		}

		adds := config.DefaultP2P().QriBootstrapAddrs

		for i, addr := range cfg.P2P.QriBootstrapAddrs {
			// remove any old, invalid addresses
			if removes[addr] {
				cfg.P2P.QriBootstrapAddrs = delIdx(i, cfg.P2P.QriBootstrapAddrs)
			}
			// remove address from list of additions if already configured
			for j, add := range adds {
				if addr == add {
					// adds = append(adds[:j], adds[j+1:]...)
					adds = delIdx(j, adds)
				}
			}
		}

		cfg.P2P.QriBootstrapAddrs = append(cfg.P2P.QriBootstrapAddrs, adds...)
	}

	cfg.Revision = 1
	return nil
}

func delIdx(i int, sl []string) []string {
	if i < len(sl)-1 {
		return append(sl[:i], sl[i+1:]...)
	}

	return sl[:i]
}
