package cmd

import (
	"context"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/gen"
	"github.com/qri-io/qri/repo/test"
	repotest "github.com/qri-io/qri/repo/test"
)

// TestFactory is an implementation of the Factory interface for testing purposes
type TestFactory struct {
	ioes.IOStreams
	// QriRepoPath is the path to the QRI repository
	qriRepoPath string
	// IpfsFsPath is the path to the IPFS repo
	ipfsFsPath string
	// generator is a source of cryptographic info
	generator gen.CryptoGenerator

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
func NewTestFactory() (tf TestFactory, err error) {
	repo, err := test.NewTestRepo()
	if err != nil {
		return
	}

	cfg := config.DefaultConfigForTesting().Copy()
	tnode, err := p2p.NewTestableQriNode(repo, cfg.P2P)
	if err != nil {
		return
	}

	return TestFactory{
		IOStreams:   ioes.NewDiscardIOStreams(),
		qriRepoPath: "",
		ipfsFsPath:  "",
		generator:   repotest.NewTestCrypto(),

		repo:   repo,
		rpc:    nil,
		config: cfg,
		node:   tnode.(*p2p.QriNode),
		inst:   lib.NewInstanceFromConfigAndNode(cfg, tnode.(*p2p.QriNode)),
	}, nil
}

// NewTestFactoryInstanceOptions is an experimental test factory that allows
// instance configuration overrides
// TODO (b5) - I'm not confident this works perfectly at the moment. Let's add
// more tests to lib.NewInstance before using everywhere
func NewTestFactoryInstanceOptions(opts ...lib.Option) (tf TestFactory, err error) {
	repo, err := test.NewTestRepo()
	if err != nil {
		return
	}

	cfg := config.DefaultConfigForTesting().Copy()
	tnode, err := p2p.NewTestableQriNode(repo, cfg.P2P)
	if err != nil {
		return
	}

	opts = append([]lib.Option{
		lib.OptConfig(cfg),
		lib.OptQriNode(tnode.(*p2p.QriNode)),
	}, opts...)

	inst, err := lib.NewInstance(context.Background(), "repo", opts...)
	if err != nil {
		return TestFactory{}, err
	}

	return TestFactory{
		IOStreams:   ioes.NewDiscardIOStreams(),
		qriRepoPath: "",
		ipfsFsPath:  "",
		generator:   repotest.NewTestCrypto(),

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

// IpfsFsPath returns from internal state
func (t TestFactory) IpfsFsPath() string {
	return t.ipfsFsPath
}

// QriRepoPath returns from internal state
func (t TestFactory) QriRepoPath() string {
	return t.qriRepoPath
}

// CryptoGenerator
func (t TestFactory) CryptoGenerator() gen.CryptoGenerator {
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

// ExportRequests generates a lib.ExportRequests from internal state
func (t TestFactory) ExportRequests() (*lib.ExportRequests, error) {
	return lib.NewExportRequests(t.node, t.rpc), nil
}

// PeerRequests generates a lib.PeerRequests from internal state
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

// RenderRequests generates a lib.RenderRequests from internal state
func (t TestFactory) RenderRequests() (*lib.RenderRequests, error) {
	return lib.NewRenderRequests(t.repo, t.rpc), nil
}

func TestEnvPathFactory(t *testing.T) {
	//Needed to clean up changes after the test has finished running
	prevQRIPath := os.Getenv("QRI_PATH")
	prevIPFSPath := os.Getenv("IPFS_PATH")

	defer func() {
		os.Setenv("QRI_PATH", prevQRIPath)
		os.Setenv("IPFS_PATH", prevIPFSPath)
	}()

	//Test variables
	emptyPath := ""
	fakePath := "fake_path"
	home, err := homedir.Dir()
	if err != nil {
		t.Fatalf("Failed to find the home directory: %s", err.Error())
	}

	tests := []struct {
		qriPath    string
		ipfsPath   string
		qriAnswer  string
		ipfsAnswer string
	}{
		{emptyPath, emptyPath, filepath.Join(home, ".qri"), filepath.Join(home, ".ipfs")},
		{emptyPath, fakePath, filepath.Join(home, ".qri"), fakePath},
		{fakePath, emptyPath, fakePath, filepath.Join(home, ".ipfs")},
		{fakePath, fakePath, fakePath, fakePath},
	}

	for i, test := range tests {
		err := os.Setenv("QRI_PATH", test.qriPath)
		if err != nil {
			t.Errorf("case %d failed to set up QRI_PATH: %s", i, err.Error())
		}

		err = os.Setenv("IPFS_PATH", test.ipfsPath)
		if err != nil {
			t.Errorf("case %d failed to set up IPFS_PATH: %s", i, err.Error())
		}

		qriResult, ipfsResult := EnvPathFactory()

		if !strings.EqualFold(qriResult, test.qriAnswer) {
			t.Errorf("case %d expected qri path to be %s, but got %s", i, test.qriAnswer, qriResult)
		}

		if !strings.EqualFold(ipfsResult, test.ipfsAnswer) {
			t.Errorf("case %d Expected ipfs path to be %s, but got %s", i, test.ipfsAnswer, ipfsResult)
		}

	}
}
