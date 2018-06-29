package cmd

import (
	"net/rpc"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/test"
	"github.com/qri-io/registry/regclient"
)

// TestFactory is an implementation of the Factory interface for testing purposes
type TestFactory struct {
	IOStreams
	// QriRepoPath is the path to the QRI repository
	qriRepoPath string
	// IpfsFsPath is the path to the IPFS repo
	ipfsFsPath string

	// Configuration object
	config *config.Config
	node   *p2p.QriNode
	repo   repo.Repo
	rpc    *rpc.Client
}

// NewTestFactory creates TestFactory object with an in memory test repo
// with an optional registry client. In tests users can create mock registry
// servers and pass in a client connected to that mock, or omit the registry
// client entirely for testing without a designated registry
func NewTestFactory(c *regclient.Client) (tf TestFactory, err error) {
	repo, err := test.NewTestRepo(c)
	if err != nil {
		return
	}

	cfg := config.DefaultConfig()

	return TestFactory{
		qriRepoPath: "",
		ipfsFsPath:  "",

		repo:   repo,
		rpc:    nil,
		config: cfg,
		node:   nil,
	}, nil
}

// Config returns from internal state
func (t TestFactory) Config() (*config.Config, error) {
	return t.config, nil
}

// IpfsFsPath returns from internal state
func (t TestFactory) IpfsFsPath() string {
	return t.ipfsFsPath
}

// QriRepoPath returns from internal state
func (t TestFactory) QriRepoPath() string {
	return t.qriRepoPath
}

// Repo returns from internal state
func (t TestFactory) Repo() (repo.Repo, error) {
	return t.repo, nil
}

// RPC returns from internal state
func (t TestFactory) RPC() *rpc.Client {
	return nil
}

// DatasetRequests generates a lib.DatasetRequests from internal state
func (t TestFactory) DatasetRequests() (*lib.DatasetRequests, error) {
	return lib.NewDatasetRequests(t.repo, t.rpc), nil
}

// RegistryRequests generates a lib.RegistryRequests from internal state
func (t TestFactory) RegistryRequests() (*lib.RegistryRequests, error) {
	return lib.NewRegistryRequests(t.repo, t.rpc), nil
}

// HistoryRequests generates a lib.HistoryRequests from internal state
func (t TestFactory) HistoryRequests() (*lib.HistoryRequests, error) {
	return lib.NewHistoryRequests(t.repo, t.rpc), nil
}

// PeerRequests generates a lib.PeerRequests from internal state
func (t TestFactory) PeerRequests() (*lib.PeerRequests, error) {
	return lib.NewPeerRequests(nil, t.rpc), nil
}

// ProfileRequests generates a lib.ProfileRequests from internal state
func (t TestFactory) ProfileRequests() (*lib.ProfileRequests, error) {
	return lib.NewProfileRequests(t.repo, t.rpc), nil
}

// SelectionRequests creates a lib.SelectionRequests from internal state
func (t TestFactory) SelectionRequests() (*lib.SelectionRequests, error) {
	return lib.NewSelectionRequests(t.repo, t.rpc), nil
}

// SearchRequests generates a lib.SearchRequests from internal state
func (t TestFactory) SearchRequests() (*lib.SearchRequests, error) {
	return lib.NewSearchRequests(t.repo, t.rpc), nil
}

// RenderRequests generates a lib.RenderRequests from internal state
func (t TestFactory) RenderRequests() (*lib.RenderRequests, error) {
	return lib.NewRenderRequests(t.repo, t.rpc), nil
}
