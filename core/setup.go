package core

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/registry/regclient"
)

// ErrHandleTaken is for when a peername is already taken
var ErrHandleTaken = fmt.Errorf("handle is taken")

// SetupParams encapsulates arguments for Setup
type SetupParams struct {
	Config              *config.Config
	QriRepoPath         string
	ConfigFilepath      string
	SetupIPFS           bool
	IPFSFsPath          string
	SetupIPFSConfigData []byte
}

// Setup provisions a new qri instance, it intentionally doesn't conform to the RPC function signature
// because remotely invoking setup doesn't make much sense
func Setup(p SetupParams) error {
	cfg := p.Config
	if cfg.Profile == nil {
		cfg.Profile = config.DefaultProfile()
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %s", err.Error())
	}

	if cfg.Registry != nil {
		privkey, err := cfg.Profile.DecodePrivateKey()
		if err != nil {
			return err
		}

		reg := regclient.NewClient(&regclient.Config{
			Location: cfg.Registry.Location,
		})

		if err := reg.PutProfile(cfg.Profile.Peername, privkey); err != nil {
			if strings.Contains(err.Error(), "taken") {
				return ErrHandleTaken
			}
			return err
		}
	}

	if err := os.MkdirAll(p.QriRepoPath, os.ModePerm); err != nil {
		return fmt.Errorf("error creating home dir: %s", err.Error())
	}

	if p.SetupIPFS {
		tmpIPFSConfigPath := ""
		if p.SetupIPFSConfigData != nil {
			// TODO - remove this temp file & instead adjust ipfs.InitRepo to accept an io.Reader
			tmpIPFSConfigPath = filepath.Join(os.TempDir(), "ipfs_init_config")

			if err := ioutil.WriteFile(tmpIPFSConfigPath, p.SetupIPFSConfigData, os.ModePerm); err != nil {
				return err
			}

			defer func() {
				os.Remove(tmpIPFSConfigPath)
			}()
		}

		if err := ipfs.InitRepo(p.IPFSFsPath, tmpIPFSConfigPath); err != nil {
			if !strings.Contains(err.Error(), "already") {
				return fmt.Errorf("error creating IPFS repo: %s", err.Error())
			}
		}

		if _, err := os.Stat(p.IPFSFsPath); os.IsNotExist(err) {
			return fmt.Errorf("no IPFS repo exists at %s, things aren't going to work properly", p.IPFSFsPath)
		}
	}

	if err := cfg.WriteToFile(p.ConfigFilepath); err != nil {
		return fmt.Errorf("error writing config: %s", err.Error())
	}

	Config = cfg
	ConfigFilepath = p.ConfigFilepath
	return nil
}

// Teardown reverses the setup process, destroying a user's privateKey
// and removing local qri data
func Teardown() error {
	return nil
}
