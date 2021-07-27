package regclient

import (
	"context"

	"github.com/qri-io/dataset"
	qhttp "github.com/qri-io/qri/lib/http"
	"github.com/qri-io/qri/registry"
)

// GetFollowing returns a list of datasets a user follows from the registry
func (c Client) GetFollowing(ctx context.Context, p *registry.FollowGetParams) ([]*dataset.Dataset, error) {
	if c.httpClient == nil {
		return nil, ErrNoRegistry
	}

	results := []*dataset.Dataset{}
	err := c.httpClient.Call(ctx, qhttp.AERegistryGetFollowing, "", p, results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// Follow updates the users follow status for a datasets on the registry
func (c Client) Follow(ctx context.Context, p *registry.FollowParams) error {
	if c.httpClient == nil {
		return ErrNoRegistry
	}
	return c.httpClient.Call(ctx, qhttp.AERegistryFollow, "", p, nil)
}
