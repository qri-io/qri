package cmd

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
)

// Factory is an interface for providing required structures to cobra commands
// It's main implementation is QriOptions
type Factory interface {
	Instance() (*lib.Instance, error)
	Config() (*config.Config, error)

	// path to qri data directory
	RepoPath() string
	CryptoGenerator() key.CryptoGenerator

	Init() error
	HTTPClient() *lib.HTTPClient
	ConnectionNode() (*p2p.QriNode, error)
}

// StandardRepoPath returns qri paths based on the QRI_PATH environment
// variable falling back to the default: $HOME/.qri
func StandardRepoPath() string {
	qriRepoPath := os.Getenv("QRI_PATH")
	if qriRepoPath == "" {
		home, err := homedir.Dir()
		if err != nil {
			panic(err)
		}
		qriRepoPath = filepath.Join(home, ".qri")
	}

	return qriRepoPath
}
