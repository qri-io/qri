package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/qfs/qipfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/gen"
)

// QriRepoExists returns nil if a qri repo is defined at the given path
// does not attempt to open the repo
func QriRepoExists(path string) error {
	// for now this just checks for an existing config file
	_, err := os.Stat(filepath.Join(path, "config.yaml"))
	if !os.IsNotExist(err) {
		return nil
	}
	return repo.ErrNoRepo
}

// SetupParams encapsulates arguments for Setup
type SetupParams struct {
	// a configuration is required. defaults to config.DefaultConfig()
	Config *config.Config
	// where to initialize qri repository
	RepoPath string
	// submit new username to the configured registry
	Register bool
	// overwrite any existing repo, erasing all data and deleting private keys
	// this is almost always a bad idea
	Overwrite bool
	// attempt to setup an IFPS repo
	SetupIPFS           bool
	SetupIPFSConfigData []byte
	// setup requires a crypto source
	Generator gen.CryptoGenerator
}

// Setup provisions a new qri instance, it intentionally doesn't conform to the
// RPC function signature because remotely invoking setup doesn't make sense
func Setup(p SetupParams) error {
	if err := QriRepoExists(p.RepoPath); err == nil && !p.Overwrite {
		return fmt.Errorf("repo already initialized")
	}

	cfg := p.Config
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	if cfg.P2P == nil {
		cfg.P2P = config.DefaultP2P()
	}
	if cfg.P2P.PrivKey == "" {
		privKey, peerID := p.Generator.GeneratePrivateKeyAndPeerID()
		cfg.P2P.PrivKey = privKey
		cfg.P2P.PeerID = peerID
	}
	if cfg.Profile == nil {
		cfg.Profile = config.DefaultProfile()
	}
	if cfg.Profile.PrivKey == "" {
		cfg.Profile.PrivKey = cfg.P2P.PrivKey
		cfg.Profile.ID = cfg.P2P.PeerID
		cfg.Profile.Peername = p.Generator.GenerateNickname(cfg.P2P.PeerID)
	}

	if err := setup(p.RepoPath, p.Config, p.Register); err != nil {
		return err
	}

	if p.SetupIPFS {
		ipfsPath := filepath.Join(p.RepoPath, "ipfs")
		// IPFS plugins need to be loaded
		if err := qipfs.LoadIPFSPluginsOnce(ipfsPath); err != nil {
			return err
		}

		if err := initIPFS(ipfsPath, p.SetupIPFSConfigData, p.Generator); err != nil {
			return err
		}
	}

	return nil
}

// TeardownParams encapsulates arguments for Setup
type TeardownParams struct {
	Config         *config.Config
	RepoPath       string
	ConfigFilepath string
}

// Teardown reverses the setup process, destroying a user's privateKey
// and removing local qri data
func Teardown(p TeardownParams) error {
	return os.RemoveAll(p.RepoPath)
}

// setup provisions a new qri instance
func setup(repoPath string, cfg *config.Config, register bool) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %s", err.Error())
	}

	// TODO (b5) - we'll need to add a choose-password-for-qri-cloud to do
	// signup for this
	// if register && cfg.Registry != nil {
	// 	pro, err := profile.NewProfile(cfg.Profile)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	reg := regclient.NewClient(&regclient.Config{
	// 		Location: cfg.Registry.Location,
	// 	})

	// 	if _, err := reg.PutProfile(pro, pro.PrivKey); err != nil {
	// 		return err
	// 	}
	// }

	if err := os.MkdirAll(repoPath, os.ModePerm); err != nil {
		return fmt.Errorf("error creating qri repo directory: %s, path: %s", err.Error(), repoPath)
	}

	cfgPath := filepath.Join(repoPath, "config.yaml")

	if err := cfg.WriteToFile(cfgPath); err != nil {
		return fmt.Errorf("error writing config: %s", err.Error())
	}
	return nil
}

// initIPFS initializes an IPFS repo
func initIPFS(path string, cfgData []byte, g gen.CryptoGenerator) error {
	tmpIPFSConfigPath := ""
	if cfgData != nil {
		// TODO - remove this temp file & instead adjust ipfs.InitRepo to accept an io.Reader
		tmpIPFSConfigPath = filepath.Join(os.TempDir(), "ipfs_init_config")

		if err := ioutil.WriteFile(tmpIPFSConfigPath, cfgData, os.ModePerm); err != nil {
			return err
		}

		defer func() {
			os.Remove(tmpIPFSConfigPath)
		}()
	}

	if err := g.GenerateEmptyIpfsRepo(path, tmpIPFSConfigPath); err != nil {
		if !strings.Contains(err.Error(), "already") {
			return fmt.Errorf("error creating IPFS repo: %s", err.Error())
		}
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("no IPFS repo exists at %s, things aren't going to work properly", path)
	}
	return nil
}
