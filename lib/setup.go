package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/qfs/qipfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo/gen"
)

// SetupParams encapsulates arguments for Setup
type SetupParams struct {
	Config              *config.Config
	QriRepoPath         string
	ConfigFilepath      string
	SetupIPFS           bool
	Register            bool
	IPFSFsPath          string
	SetupIPFSConfigData []byte
	Generator           gen.CryptoGenerator
}

// Setup provisions a new qri instance, it intentionally doesn't conform to the RPC function signature
// because remotely invoking setup doesn't make much sense
func Setup(p SetupParams) error {
	if err := setup(p.QriRepoPath, p.ConfigFilepath, p.Config, p.Register); err != nil {
		return err
	}

	if p.SetupIPFS {
		// IPFS plugins need to be loaded
		if err := qipfs.LoadIPFSPluginsOnce(p.IPFSFsPath); err != nil {
			return err
		}

		if err := initIPFS(p.IPFSFsPath, p.SetupIPFSConfigData, p.Generator); err != nil {
			return err
		}
	}

	// Config = p.Config
	// ConfigFilepath = p.ConfigFilepath
	return nil
}

// TeardownParams encapsulates arguments for Setup
type TeardownParams struct {
	Config         *config.Config
	QriRepoPath    string
	ConfigFilepath string
	// IPFSFsPath          string
}

// Teardown reverses the setup process, destroying a user's privateKey
// and removing local qri data
func Teardown(p TeardownParams) error {
	return os.RemoveAll(p.QriRepoPath)
}

// setup provisions a new qri instance
func setup(repoPath, cfgPath string, cfg *config.Config, register bool) error {
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
