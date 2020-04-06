// Package resolver centralizes logic for resolving dataset names. It provides a low-dependency
// interface that can be used throughout the stack, and also an implementation that ties together
// multiple subsystems to be used as a high-level mechanism for name resolution.
package resolver

import (
	"fmt"

	"github.com/qri-io/qri/dsref"
)

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
