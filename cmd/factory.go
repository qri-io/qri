package cmd

import (
	"net/rpc"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo/gen"
)

// Factory is an interface for providing required structures to cobra commands
// It's main implementation is QriOptions
type Factory interface {
	Instance() *lib.Instance
	Config() (*config.Config, error)

	// path to qri data directory
	RepoPath() string
	CryptoGenerator() gen.CryptoGenerator

	Init() error
	RPC() *rpc.Client
	ConnectionNode() (*p2p.QriNode, error)

	ConfigMethods() (*lib.ConfigMethods, error)
	DatasetMethods() (*lib.DatasetMethods, error)
	RemoteMethods() (*lib.RemoteMethods, error)
	RegistryClientMethods() (*lib.RegistryClientMethods, error)
	LogMethods() (*lib.LogMethods, error)
	PeerMethods() (*lib.PeerMethods, error)
	ProfileMethods() (*lib.ProfileMethods, error)
	SearchMethods() (*lib.SearchMethods, error)
	SQLMethods() (*lib.SQLMethods, error)
	FSIMethods() (*lib.FSIMethods, error)
	RenderMethods() (*lib.RenderMethods, error)
	TransformMethods() (*lib.TransformMethods, error)
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
