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

	QriPath() string
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

	// TODO (b5) - these should be deprecated:
	ExportRequests() (*lib.ExportRequests, error)
}

// StandardQriPath returns qri paths based on the QRI_PATH environment variable
// falling back to the default: $HOME/.qri
func StandardQriPath() string {
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

// RootedQriPath gives a "/qri" directory from a given root path
func RootedQriPath(root string) string {
	return filepath.Join(root, "qri")
}
