package lib

import (
	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/config"
)

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
	if err := actions.Setup(p.QriRepoPath, p.ConfigFilepath, p.Config); err != nil {
		return err
	}

	if p.SetupIPFS {
		if err := actions.InitIPFS(p.IPFSFsPath, p.SetupIPFSConfigData); err != nil {
			return err
		}
	}

	Config = p.Config
	ConfigFilepath = p.ConfigFilepath
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
