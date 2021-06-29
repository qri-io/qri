package lib

import (
	"context"

	"github.com/qri-io/qri/config/feature"
)

// FeatureFlagMethods encapsulates business logic for managing
// feature flags
type FeatureFlagMethods struct {
	d dispatcher
}

// Name returns the name of this method group
func (m FeatureFlagMethods) Name() string {
	return "feature"
}

// Attributes defines attributes for each method
func (m FeatureFlagMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"list": {Endpoint: AEListFlags, HTTPVerb: "POST"},
		"get":  {Endpoint: AEGetFlag, HTTPVerb: "POST"},
		"put":  {Endpoint: AESetFlag, HTTPVerb: "POST"},
	}
}

// List returns a list of all the feature flags
func (m FeatureFlagMethods) List(ctx context.Context, p *EmptyParams) ([]*feature.Flag, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "list"), p)
	if res, ok := got.([]*feature.Flag); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Get returns a feature flag based on the feature.ID in params
func (m FeatureFlagMethods) Get(ctx context.Context, p *feature.Flag) (*feature.Flag, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "get"), p)
	if res, ok := got.(*feature.Flag); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Put updates a feature flag based on the feature.ID and active status in params
func (m FeatureFlagMethods) Put(ctx context.Context, p *feature.Flag) (*feature.Flag, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "put"), p)
	if res, ok := got.(*feature.Flag); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// featureFlagsImpl holds the method implementations for FeatureFlagMethods
type featureFlagsImpl struct{}

// List returns a list of all the feature flags
func (m featureFlagsImpl) List(scope scope, p *EmptyParams) ([]*feature.Flag, error) {
	return scope.FeatureFlags().List(scope.Context())
}

// Get returns a feature flag based on the feature.ID in params
func (m featureFlagsImpl) Get(scope scope, p *feature.Flag) (*feature.Flag, error) {
	return scope.FeatureFlags().Get(scope.Context(), p.ID)
}

// Put updates a feature flag based on the feature.ID and active status in params
func (m featureFlagsImpl) Put(scope scope, p *feature.Flag) (*feature.Flag, error) {
	return scope.FeatureFlags().Put(scope.Context(), p)
}
