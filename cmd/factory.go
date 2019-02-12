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
	Config() (*config.Config, error)

	IpfsFsPath() string
	QriRepoPath() string
	CryptoGenerator() gen.CryptoGenerator

	Init() error
	RPC() *rpc.Client
	ConnectionNode() (*p2p.QriNode, error)

	Qri() (*lib.Qri, error)

	// DatasetRequests() (*lib.DatasetRequests, error)
	// RegistryRequests() (*lib.RegistryRequests, error)
	// LogRequests() (*lib.LogRequests, error)
	// ExportRequests() (*lib.ExportRequests, error)
	// PeerRequests() (*lib.PeerRequests, error)
	// ProfileRequests() (*lib.ProfileRequests, error)
	// SearchRequests() (*lib.SearchRequests, error)
	// RenderRequests() (*lib.RenderRequests, error)
	// SelectionRequests() (*lib.SelectionRequests, error)
}

// PathFactory is a function that returns paths to qri & ipfs repos
type PathFactory func() (string, string)

// EnvPathFactory returns qri & IPFS paths based on enviornment variables
// falling back to $HOME/.qri && $HOME/.ipfs
func EnvPathFactory() (string, string) {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	qriRepoPath := os.Getenv("QRI_PATH")
	if qriRepoPath == "" {
		qriRepoPath = filepath.Join(home, ".qri")
	}

	ipfsFsPath := os.Getenv("IPFS_PATH")
	if ipfsFsPath == "" {
		ipfsFsPath = filepath.Join(home, ".ipfs")
	}
	return qriRepoPath, ipfsFsPath
}

// NewDirPathFactory creates a path factory that sets qri & ipfs paths to
// dir/qri & qri/ipfs
func NewDirPathFactory(dir string) PathFactory {
	return func() (string, string) {
		return filepath.Join(dir, "qri"), filepath.Join(dir, "ipfs")
	}
}
