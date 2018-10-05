// Package lib implements core qri business logic. It exports
// canonical methods that a qri instance can perform regardless of
// client interface. API's of any sort must use lib methods
package lib

import (
	"encoding/gob"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/p2p"
)

var log = golog.Logger("lib")

// VersionNumber is the current version qri
const VersionNumber = "0.5.5"

// Requests defines a set of library methods
type Requests interface {
	// CoreRequestsName confirms participation in the CoreRequests interface while
	// also giving a human readable string for logging purposes
	CoreRequestsName() string
}

func init() {
	// Fields like dataset.Structure.Schema contain data of arbitrary types,
	// registering with the gob package prevents errors when sending them
	// over net/rpc calls.
	gob.Register([]interface{}{})
	gob.Register(map[string]interface{}{})
}

// Receivers returns a slice of CoreRequests that defines the full local
// API of lib methods
func Receivers(node *p2p.QriNode) []Requests {
	return []Requests{
		NewDatasetRequests(node, nil),
		NewRegistryRequests(node, nil),
		NewLogRequests(node, nil),
		NewPeerRequests(node, nil),
		NewProfileRequests(node, nil),
		NewSearchRequests(node, nil),
		NewRenderRequests(node.Repo, nil),
		NewSelectionRequests(node.Repo, nil),
	}
}
