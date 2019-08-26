package lib

import (
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/p2p"
)

// RegistryRequests defines business logic for working with registries
// TODO (b5): switch to using an Instance instead of separate fields
type RegistryRequests struct {
	node *p2p.QriNode
	cli  *rpc.Client
}

// CoreRequestsName implements the Requests interface
func (RegistryRequests) CoreRequestsName() string { return "registry" }

// NewRegistryRequests creates a RegistryRequests pointer from either a repo
// or an rpc.Client
func NewRegistryRequests(node *p2p.QriNode, cli *rpc.Client) *RegistryRequests {
	if node != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewRegistryRequests"))
	}

	return &RegistryRequests{
		node: node,
		cli:  cli,
	}
}