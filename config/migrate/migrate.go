// Package migrate defines migrations for qri configuration files
package migrate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/mitchellh/go-homedir"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs/qipfs"
	"github.com/qri-io/qri/config"
	qerr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/buildrepo"
)

var (
	log = logging.Logger("migrate")
	// ErrNeedMigration indicates a migration is required
	ErrNeedMigration = fmt.Errorf("migration required")
	// ErrMigrationSucceeded indicates a migration completed executing
	ErrMigrationSucceeded = errors.New("migration succeeded")
)

// RunMigrations executes migrations. if a migration is required, the shouldRun
// func is called, and exits without migrating if shouldRun returns false.
// if errorOnSuccess is true, a completed migration will return
// ErrMigrationSucceeded instead of nil
func RunMigrations(streams ioes.IOStreams, cfg *config.Config, shouldRun func() bool, errorOnSuccess bool) (err error) {
	if cfg.Revision != config.CurrentConfigRevision {
		if !shouldRun() {
			return qerr.New(ErrNeedMigration, `your repo requires migration before it can run`)
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

		if errorOnSuccess {
			return ErrMigrationSucceeded
		}
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
	newIPFSPath := filepath.Join(qriPath, "ipfs")
	oldIPFSPath := configVersionOneIPFSPath()

	// TODO(ramfox): qfs migration
	if err := qipfs.InternalizeIPFSRepo(oldIPFSPath, newIPFSPath); err != nil {
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

	if err := maybeRemoveIPFSRepo(cfg, oldIPFSPath); err != nil {
		log.Debug(err)
		fmt.Printf("error removing IPFS repo at %q:\n\t%s", oldIPFSPath, err)
		fmt.Printf(`qri has successfully internalized this IPFS repo, and no longer 
		needs the folder at %q. you may want to remove it
`, oldIPFSPath)
	}

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

// In qri v0.9.8 & earlier, the IPFS path location was determined by the
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

func maybeRemoveIPFSRepo(cfg *config.Config, oldPath string) error {
	fmt.Println("\nChecking if existing IPFS directory contains non-qri data...")
	repoPath := filepath.Dir(cfg.Path())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)

	r, err := buildrepo.New(ctx, repoPath, cfg)
	if err != nil {
		cancel()
		return err
	}

	defer gracefulShutdown(cancel, r)

	// Note: this is intentionally using the new post-migration IPFS repo to judge
	// pin presence, because we can't operate on the old one
	fs := r.Filesystem().Filesystem(qipfs.FilestoreType)
	if fs == nil {
		return nil
	}

	logbookPaths, err := r.Logbook().AllReferencedDatasetPaths(ctx)
	if err != nil {
		return err
	}

	paths := map[string]struct{}{
		// add common paths auto-added on IPFS init we can safely ignore
		"/ipld/QmQPeNsJPyVWPFDVHb77w8G42Fvo15z4bG2X8D2GhfbSXc": {},
		"/ipld/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn": {},
		"/ipld/QmS4ustL54uo8FzR9455qaxZwuMiUhyvMcX9Ba8nUH4uVv": {},
	}

	for p := range logbookPaths {
		// switch "/ipfs/" prefixes for "/ipld/"
		p = strings.Replace(p, "/ipfs", "/ipld", 1)
		paths[p] = struct{}{}
	}

	unknownPinCh, err := fs.(*qipfs.Filestore).PinsetDifference(ctx, paths)
	if err != nil {
		return err
	}

	log.Debugf("checking pins...%#v\n", paths)

	unknown := []string{}
	for path := range unknownPinCh {
		log.Debugf("checking if unknown pin is a dataset: %s\n", path)
		path = strings.Replace(path, "/ipld", "/ipfs", 1)
		// check if the pinned path is a valid qri dataset, looking for "dataset.json"
		// this check allows us to ignore qri data logbook doesn't know about
		if f, err := fs.Get(ctx, fmt.Sprintf("%s/dataset.json", path)); err == nil {
			f.Close()
		} else {
			unknown = append(unknown, path)
		}
	}

	if len(unknown) > 0 {
		fmt.Printf(`
Qri left your original IPFS repo in place because it contains pinned data that 
Qri isn't managing. Qri has created an internal copy of your IPFS repo, and no
longer requires the repo at %q
`, oldPath)
		if len(unknown) < 10 {
			fmt.Printf("unknown pins:\n\t%s\n\n", strings.Join(unknown, "\n\t"))
		} else {
			fmt.Printf("\nfound %d unknown pins\n\n", len(unknown))
		}
	} else {
		if err := os.RemoveAll(oldPath); err != nil {
			return err
		}
		fmt.Printf("moved IPFS repo from %q into qri repo\n", oldPath)
	}

	log.Info("successfully migrated repo, shutting down")

	return nil
}

func gracefulShutdown(cancel context.CancelFunc, r repo.Repo) {
	var wg sync.WaitGroup
	go func() {
		<-r.Done()
		wg.Done()
	}()

	wg.Add(1)
	cancel()
	wg.Wait()
}
