// Package resolve centralizes logic for resolving dataset names. It provides a
// low-dependency interface that can be used throughout the stack, and also an
// implementation that ties together multiple subsystems to be used as a
// high-level mechanism for name resolution.
package resolve

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/dsref"
)

// RefResolver finds the identifier and HEAD path for a dataset reference
type RefResolver interface {
	// ResolveRef uses ref as an outParam, setting ref.ID and ref.Path on success
	// some implementations of name resolution may make network calls
	ResolveRef(ctx context.Context, ref *dsref.Ref) error
}

// Resolver resolves identifiers into info about datasets
type Resolver interface {
	// GetInfo takes an initID for a dataset and returns the most recent VersionInfo
	GetInfo(initID string) *dsref.VersionInfo
	// GetInfoByDsref takes a dsref and returns the most recent VersionInfo. GetInfo should be
	// used instead when possible.
	GetInfoByDsref(dr dsref.Ref) *dsref.VersionInfo
}

// ErrCannotResolveName is an error representing common name resolution problems
var ErrCannotResolveName = fmt.Errorf("cannot resolve name")
