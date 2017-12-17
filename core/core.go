// Package core implements core qri business logic. It exports
// canonical methods that a qri instance can perform regardless of
// client interface. API's of any sort must use core methods.
// Tests of core functions should be extensive.
// Refactoring core methods should be a process of moving logic
// out of core itself in favor of delegation to logical subsystems.
package core

import (
	"github.com/qri-io/qri/p2p"
)

// Requests defines a set of core methods
type Requests interface {
	// CoreRequestsName confirms participation in the CoreRequests interface while
	// also giving a human readable string for logging purposes
	CoreRequestsName() string
}

// Receivers returns a slice of CoreRequests that defines the full local
// API of core methods
func Receivers(node *p2p.QriNode) []CoreRequests {
	r := node.Repo
	return []CoreRequests{
		NewDatasetRequests(r, nil),
		NewHistoryRequests(r, nil),
		NewPeerRequests(node, nil),
		NewProfileRequests(r, nil),
		NewQueryRequests(r, nil),
		NewSearchRequests(r, nil),
	}
}
