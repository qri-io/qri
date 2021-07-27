package lib

import (
	"context"

	"github.com/qri-io/dataset"
	qhttp "github.com/qri-io/qri/lib/http"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/repo"
)

// FollowMethods groups together methods for follows
type FollowMethods struct {
	d dispatcher
}

// Name returns the name of this method group
func (m FollowMethods) Name() string {
	return "follow"
}

// Attributes defines attributes for each method
func (m FollowMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"get":    {Endpoint: qhttp.AERegistryGetFollowing, HTTPVerb: "POST"},
		"follow": {Endpoint: qhttp.AERegistryFollow, HTTPVerb: "POST"},
	}
}

// Get returns a list of datasets a user follows
func (m FollowMethods) Get(ctx context.Context, p *registry.FollowGetParams) ([]*dataset.Dataset, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "get"), p)
	if res, ok := got.([]*dataset.Dataset); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Follow updates the follow status of the current user for a given dataset
func (m FollowMethods) Follow(ctx context.Context, p *registry.FollowParams) error {
	_, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "follow"), p)
	return dispatchReturnError(nil, err)
}

// followImpl holds the method implementations for follows
type followImpl struct{}

// Get returns a list of datasets a user follows
func (followImpl) Get(scope scope, p *registry.FollowGetParams) ([]*dataset.Dataset, error) {
	client := scope.RegistryClient()
	if client == nil {
		return nil, repo.ErrNoRegistry
	}

	regResults, err := client.GetFollowing(scope.Context(), p)
	if err != nil {
		return nil, err
	}

	return regResults, nil
}

// Follow updates the follow status of the current user for a given dataset
func (m followImpl) Follow(scope scope, p *registry.FollowParams) error {
	client := scope.RegistryClient()
	if client == nil {
		return repo.ErrNoRegistry
	}

	return client.Follow(scope.Context(), p)
}
