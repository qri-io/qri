// Package regserver provides a mock registry server for testing purposes
package regserver

import (
	"context"
	"net/http/httptest"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/remote/registry"
	"github.com/qri-io/qri/remote/registry/regclient"
	"github.com/qri-io/qri/remote/registry/regserver/handlers"
	"github.com/qri-io/qri/repo"
	repotest "github.com/qri-io/qri/repo/test"
)

// TODO (b5) - this value is used all over the plcae need a better strategy for
const registryPeerID = "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"

func init() {
	// don't need verbose logging when working with mock servers
	handlers.SetLogLevel("error")
}

// NewMockServer creates an in-memory mock server & matching registry client
func NewMockServer() (*regclient.Client, *httptest.Server) {
	return NewMockServerRegistry(NewMemRegistry(nil))
}

// NewMockServerRegistry creates a mock server & client with a passed-in registry
func NewMockServerRegistry(reg registry.Registry) (*regclient.Client, *httptest.Server) {
	s := httptest.NewServer(handlers.NewRoutes(reg))
	c := regclient.NewClient(&regclient.Config{Location: s.URL})
	return c, s
}

// NewMemRegistry creates a new in-memory registry
func NewMemRegistry(rem *remote.Remote) registry.Registry {
	return registry.Registry{
		Remote:   rem,
		Profiles: registry.NewMemProfiles(),
	}
}

// NewTempRegistry creates a functioning registry with a teardown function
// TODO(b5) - the tempRepo.Repo call in this func *requires* the passed-in
// context be cancelled at some point. drop the cleanup function return in
// favour of listening for ctx.Done and running the cleanup routine internally
func NewTempRegistry(ctx context.Context, peername, tmpDirPrefix string, g key.CryptoGenerator) (*registry.Registry, func(), error) {
	tempRepo, err := repotest.NewTempRepo(peername, tmpDirPrefix, g)
	if err != nil {
		return nil, nil, err
	}

	r, err := tempRepo.Repo(ctx)
	if err != nil {
		return nil, nil, err
	}

	teardown := func() {
		<-r.Done()
		tempRepo.Delete()
	}

	p2pCfg := config.DefaultP2P()
	p2pCfg.PeerID = registryPeerID

	localResolver := dsref.SequentialResolver(r.Dscache(), r)
	node, err := p2p.NewQriNode(r, p2pCfg, r.Bus(), localResolver)
	if err != nil {
		return nil, nil, err
	}

	remoteCfg := &config.Remote{
		Enabled:          true,
		AcceptSizeMax:    -1,
		AcceptTimeoutMs:  -1,
		RequireAllBlocks: false,
		AllowRemoves:     true,
	}

	rem, err := remote.NewRemote(node, remoteCfg, node.Repo.Logbook())
	if err != nil {
		return nil, nil, err
	}

	reg := &registry.Registry{
		Remote:   rem,
		Profiles: registry.NewMemProfiles(),
		Search:   MockRepoSearch{Repo: r},
	}

	return reg, teardown, nil
}

// MockRepoSearch proxies search to base.ListDatasets' "term" argument for
// simple-but-real search
type MockRepoSearch struct {
	repo.Repo
}

// Search implements the registry.Searchable interface
func (ss MockRepoSearch) Search(p registry.SearchParams) ([]*dataset.Dataset, error) {
	ctx := context.Background()
	refs, err := base.ListDatasets(ctx, ss.Repo, p.Q, 0, 1000, false, true, false)
	if err != nil {
		return nil, err
	}

	var res []*dataset.Dataset
	for _, ref := range refs {
		res = append(res, ref.Dataset)
	}
	return res, nil
}
