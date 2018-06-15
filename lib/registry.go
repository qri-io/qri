package lib

import (
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/actions"
)

// RegistryRequests defines business logic for working with registries
type RegistryRequests struct {
	repo actions.Registry
	cli  *rpc.Client
}

// CoreRequestsName implements the Requests interface
func (RegistryRequests) CoreRequestsName() string { return "registry" }

// NewRegistryRequests creates a DatasetRequests pointer from either a repo
// or an rpc.Client
func NewRegistryRequests(r repo.Repo, cli *rpc.Client) *RegistryRequests {
	if r != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewRegistryRequests"))
	}

	return &RegistryRequests{
		repo: actions.Registry{r},
		cli:  cli,
	}
}

// Publish a dataset to a registry
func (r *RegistryRequests) Publish(ref *repo.DatasetRef, done *bool) error {
	if r.cli != nil {
		return r.cli.Call("RegistryRequests.Publish", ref, done)
	}
	return r.repo.Publish(*ref)
}

// Unpublish a dataset from a registry
func (r *RegistryRequests) Unpublish(ref *repo.DatasetRef, done *bool) error {
	if r.cli != nil {
		return r.cli.Call("RegistryRequests.Unpublish", ref, done)
	}
	return r.repo.Unpublish(*ref)
}
