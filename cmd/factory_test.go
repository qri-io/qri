package cmd

import (
	"context"
	"net/rpc"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/config"
	testcfg "github.com/qri-io/qri/config/test"
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

	cfg := testcfg.DefaultConfigForTesting().Copy()
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

	cfg := testcfg.DefaultConfigForTesting().Copy()
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

func (t TestFactory) Instance() (*lib.Instance, error) {
	return t.inst, nil
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

// HTTPClient returns nil for tests
func (t TestFactory) HTTPClient() *lib.HTTPClient {
	return nil
}
