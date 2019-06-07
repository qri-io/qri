package lib

import (
	"github.com/qri-io/qri/actions"
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
	if err := actions.Setup(p.QriRepoPath, p.ConfigFilepath, p.Config, p.Register); err != nil {
		return err
	}

	if p.SetupIPFS {
		// need to load plugins before attempting to configure IPFS, flatfs is
		// specified as part of the default IPFS configuration, but all flatfs
		// code is loaded as a plugin.  ¯\_(ツ)_/¯
		//
		// This works without anything present in the /.ipfs/plugins/ directory b/c
		// the default plugin set is complied into go-ipfs (and subsequently, the
		// qri binary) by default
		if err := loadIPFSPluginsOnce(p.IPFSFsPath); err != nil {
			return err
		}

		if err := actions.InitIPFS(p.IPFSFsPath, p.SetupIPFSConfigData, p.Generator); err != nil {
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
	return actions.Teardown(p.QriRepoPath, p.Config)
}
