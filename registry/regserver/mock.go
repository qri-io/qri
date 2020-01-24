// Package regserver provides a mock registry server for testing purposes
package regserver

import (
	"context"
	"net/http/httptest"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regclient"
	"github.com/qri-io/qri/registry/regserver/handlers"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/repo"
)

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

// MockRepoSearch proxies search to base.ListDatasets' "term" argument for
// simple-but-real search
type MockRepoSearch struct {
	repo.Repo
}

// Search implements the registry.Searchable interface
func (ss MockRepoSearch) Search(p registry.SearchParams) ([]*dataset.Dataset, error) {
	ctx := context.Background()
	refs, err := base.ListDatasets(ctx, ss.Repo, p.Q, 1000, 0, false, true, false)
	if err != nil {
		return nil, err
	}

	var res []*dataset.Dataset
	for _, ref := range refs {
		res = append(res, ref.Dataset)
	}
	return res, nil
}
