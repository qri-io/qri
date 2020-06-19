// Package migrate defines migrations for qri configuration files
package migrate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs/qipfs"
	"github.com/qri-io/qri/config"
	qerr "github.com/qri-io/qri/errors"
)

// ErrNeedMigration indicates a migration is required
var ErrNeedMigration = fmt.Errorf("migration required")

// RunMigrations checks to see if any migrations runs them
func RunMigrations(streams ioes.IOStreams, cfg *config.Config, interactive bool) (err error) {
	if cfg.Revision != config.CurrentConfigRevision {
		if interactive {
			msg := `Your repo needs updating before qri can start. 
Run migration now?`
			if !confirm(streams.Out, streams.In, msg) {
				return qerr.New(ErrNeedMigration, `your repo requires migration before it can run`)
			}
		}

		streams.PrintErr("migrating configuration...\n")
		if cfg.Revision == 0 {
			if err := ZeroToOne(cfg); err != nil {
				return err
			}
		}
		if cfg.Revision == 1 {
			if err := OneToTwo(cfg); err != nil {
				return err
			}
		}
		streams.PrintErr("done!\n")
	}
	return nil
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

	if err := safeWriteConfig(cfg); err != nil {
		rollbackConfigWrite(cfg)
		return err
	}

	return nil
}

// OneToTwo migrates a configuration from Revision 1 to Revision 2
func OneToTwo(cfg *config.Config) error {
	qriPath := filepath.Dir(cfg.Path())
	// qri repo path must be the base of the config path
	newIPFSPath := filepath.Join(qriPath, "ipfs")

	// TODO(ramfox): qfs migration
	if err := qipfs.InternalizeIPFSRepo(configVersionOneIPFSPath(), newIPFSPath); err != nil {
		return err
	}

	if err := oneToTwoConfig(cfg); err != nil {
		return err
	}
	cfg.Revision = 2
	if err := cfg.Validate(); err != nil {
		return qerr.New(err, "config is invalid")
	}
	if err := safeWriteConfig(cfg); err != nil {
		rollbackConfigWrite(cfg)
		return err
	}

	// TODO(ramfox): remove original ipfs repo after all migrations were successful
	return nil
}

func oneToTwoConfig(cfg *config.Config) error {
	if cfg.API != nil {
		apiCfg := cfg.API
		defaultAPICfg := config.DefaultAPI()
		if apiCfg.Address == "" {
			apiCfg.Address = defaultAPICfg.Address
		}
		if apiCfg.WebsocketAddress == "" {
			apiCfg.WebsocketAddress = defaultAPICfg.WebsocketAddress
		}
	} else {
		return qerr.New(fmt.Errorf("invalid config"), "config does not contain API configuration")
	}

	if cfg.RPC != nil {
		defaultRPCCfg := config.DefaultRPC()
		if cfg.RPC.Address == "" {
			cfg.RPC.Address = defaultRPCCfg.Address
		}
	} else {
		return qerr.New(fmt.Errorf("invalid config"), "config does not contain RPC configuration")
	}

	cfg.Filesystems = config.DefaultFilesystems()

	return nil
}

func rollbackConfigWrite(cfg *config.Config) {
	cfgPath := cfg.Path()
	if len(cfgPath) == 0 {
		return
	}
	tmpCfgPath := getTmpConfigFilepath(cfgPath)
	if _, err := os.Stat(tmpCfgPath); !os.IsNotExist(err) {
		os.Remove(tmpCfgPath)
	}
}

func safeWriteConfig(cfg *config.Config) error {
	cfgPath := cfg.Path()
	if len(cfgPath) == 0 {
		return qerr.New(fmt.Errorf("invalid path"), "could not determine config path")
	}
	tmpCfgPath := getTmpConfigFilepath(cfgPath)
	if err := cfg.WriteToFile(tmpCfgPath); err != nil {
		return qerr.New(err, fmt.Sprintf("could not write config to path %s", tmpCfgPath))
	}
	if err := os.Rename(tmpCfgPath, cfgPath); err != nil {
		return qerr.New(err, fmt.Sprintf("could not write config to path %s", cfgPath))
	}

	return nil
}

func getTmpConfigFilepath(cfgPath string) string {
	cfgDir := filepath.Dir(cfgPath)
	tmpCfgPath := filepath.Join(cfgDir, "config_updated.yaml")
	return tmpCfgPath
}

func delIdx(i int, sl []string) []string {
	if i < len(sl)-1 {
		return append(sl[:i], sl[i+1:]...)
	}

	return sl[:i]
}

// In qri v0.9.8 & earlier, the IPFS path location was detrimined by the
// IPFS_PATH env var, and falling back to $HOME/.ipfs.
func configVersionOneIPFSPath() string {
	path := os.Getenv("IPFS_PATH")
	if path != "" {
		return path
	}
	home, err := homedir.Dir()
	if err != nil {
		panic(fmt.Sprintf("Failed to find the home directory: %s", err.Error()))
	}
	return filepath.Join(home, ".ipfs")
}

func confirm(w io.Writer, r io.Reader, message string) bool {
	input := prompt(w, r, fmt.Sprintf("%s [Y/n]: ", message))
	if input == "" {
		return true
	}

	return (input == "y" || input == "yes")
}

func prompt(w io.Writer, r io.Reader, msg string) string {
	var input string
	fmt.Fprintf(w, msg)
	fmt.Fscanln(r, &input)
	return strings.TrimSpace(strings.ToLower(input))
}
