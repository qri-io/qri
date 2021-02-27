package cmd

import (
	"context"
	"net/rpc"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/test"
	repotest "github.com/qri-io/qri/repo/test"
)

// TestFactory is an implementation of the Factory interface for testing purposes
type TestFactory struct {
	ioes.IOStreams
	// path to qri data directory
	repoPath string
	// generator is a source of cryptographic info
	generator key.CryptoGenerator

	inst *lib.Instance
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
func NewTestFactory(ctx context.Context) (tf TestFactory, err error) {
	repo, err := test.NewTestRepo()
	if err != nil {
		return
	}

	cfg := config.DefaultConfigForTesting().Copy()
	tnode, err := p2p.NewTestableQriNode(repo, cfg.P2P, event.NilBus)
	if err != nil {
		return
	}

	return TestFactory{
		IOStreams: ioes.NewDiscardIOStreams(),
		generator: repotest.NewTestCrypto(),

		repo:   repo,
		rpc:    nil,
		config: cfg,
		node:   tnode.(*p2p.QriNode),
		inst:   lib.NewInstanceFromConfigAndNode(ctx, cfg, tnode.(*p2p.QriNode)),
	}, nil
}

// NewTestFactoryInstanceOptions is an experimental test factory that allows
// instance configuration overrides
// TODO (b5) - I'm not confident this works perfectly at the moment. Let's add
// more tests to lib.NewInstance before using everywhere
func NewTestFactoryInstanceOptions(ctx context.Context, repoPath string, opts ...lib.Option) (tf TestFactory, err error) {
	repo, err := test.NewTestRepo()
	if err != nil {
		return
	}

	cfg := config.DefaultConfigForTesting().Copy()
	tnode, err := p2p.NewTestableQriNode(repo, cfg.P2P, event.NilBus)
	if err != nil {
		return
	}

	opts = append([]lib.Option{
		lib.OptConfig(cfg),
		lib.OptQriNode(tnode.(*p2p.QriNode)),
	}, opts...)

	inst, err := lib.NewInstance(ctx, repoPath, opts...)
	if err != nil {
		return TestFactory{}, err
	}

	return TestFactory{
		IOStreams: ioes.NewDiscardIOStreams(),
		generator: repotest.NewTestCrypto(),

		repo:   repo,
		rpc:    nil,
		config: cfg,
		node:   tnode.(*p2p.QriNode),
		inst:   inst,
	}, nil
}

// Config returns from internal state
func (t TestFactory) Config() (*config.Config, error) {
	return t.config, nil
}

func (t TestFactory) Instance() *lib.Instance {
	return t.inst
}

// RepoPath returns the path to the qri directory from internal state
func (t TestFactory) RepoPath() string {
	return t.repoPath
}

// CryptoGenerator
func (t TestFactory) CryptoGenerator() key.CryptoGenerator {
	return t.generator
}

// Init will initialize the internal state
func (t TestFactory) Init() error {
	return nil
}

// Node returns the internal qri node from state
func (t TestFactory) ConnectionNode() (*p2p.QriNode, error) {
	return t.node, nil
}

// RPC returns from internal state
func (t TestFactory) RPC() *rpc.Client {
	return nil
}

// ConfigMethods generates a lib.ConfigMethods from internal state
func (t TestFactory) ConfigMethods() (*lib.ConfigMethods, error) {
	return lib.NewConfigMethods(t.inst), nil
}

// DatasetMethods generates a lib.DatasetMethods from internal state
func (t TestFactory) DatasetMethods() (*lib.DatasetMethods, error) {
	return lib.NewDatasetMethods(t.inst), nil
}

// RemoteRequests generates a lib.RemoteRequests from internal state
func (t TestFactory) RemoteMethods() (*lib.RemoteMethods, error) {
	return lib.NewRemoteMethods(t.inst), nil
}

// RegistryClientMethods generates a lib.RegistryClientMethods from internal state
func (t TestFactory) RegistryClientMethods() (*lib.RegistryClientMethods, error) {
	return lib.NewRegistryClientMethods(t.inst), nil
}

// LogMethods generates a lib.LogMethods from internal state
func (t TestFactory) LogMethods() (*lib.LogMethods, error) {
	return lib.NewLogMethods(t.inst), nil
}

// PeerMethods generates a lib.PeerMethods from internal state
func (t TestFactory) PeerMethods() (*lib.PeerMethods, error) {
	return lib.NewPeerMethods(t.inst), nil
}

// ProfileMethods generates a lib.ProfileMethods from internal state
func (t TestFactory) ProfileMethods() (*lib.ProfileMethods, error) {
	return lib.NewProfileMethods(t.inst), nil
}

// FSIMethods generates a lib.FSIMethods from internal state
func (t TestFactory) FSIMethods() (*lib.FSIMethods, error) {
	return lib.NewFSIMethods(t.inst), nil
}

// SearchMethods generates a lib.SearchMethods from internal state
func (t TestFactory) SearchMethods() (*lib.SearchMethods, error) {
	return lib.NewSearchMethods(t.inst), nil
}

// SQLMethods generates a lib.SQLhMethods from internal state
func (t TestFactory) SQLMethods() (*lib.SQLMethods, error) {
	return lib.NewSQLMethods(t.inst), nil
}

// RenderMethods generates a lib.RenderMethods from internal state
func (t TestFactory) RenderMethods() (*lib.RenderMethods, error) {
	return lib.NewRenderMethods(t.inst), nil
}

// TransformMethods generates a lib.TransformMethods from internal state
func (t TestFactory) TransformMethods() (*lib.TransformMethods, error) {
	return lib.NewTransformMethods(t.inst), nil
}
