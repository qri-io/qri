package lib

import (
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
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

// Publish a dataset to a registry
func (r *RegistryRequests) Publish(ref *repo.DatasetRef, done *bool) (err error) {
	if r.cli != nil {
		return r.cli.Call("RegistryRequests.Publish", ref, done)
	}
	return actions.Publish(r.node, *ref)
}

// Unpublish a dataset from a registry
func (r *RegistryRequests) Unpublish(ref *repo.DatasetRef, done *bool) error {
	if r.cli != nil {
		return r.cli.Call("RegistryRequests.Unpublish", ref, done)
	}
	return actions.Unpublish(r.node, *ref)
}

// Pin asks a registry to host a copy of a dataset
func (r *RegistryRequests) Pin(ref *repo.DatasetRef, done *bool) (err error) {
	if r.cli != nil {
		return r.cli.Call("RegistryRequests.Pin", ref, done)
	}
	return actions.Pin(r.node, *ref)
}

// Unpin reverses the pin process, asking a registry to stop hosting a copy of
// an already-pinned dataset
func (r *RegistryRequests) Unpin(ref *repo.DatasetRef, done *bool) error {
	if r.cli != nil {
		return r.cli.Call("RegistryRequests.Unpin", ref, done)
	}

	return actions.Unpin(r.node, *ref)
}

// RegistryListParams encapsulates arguments to the publish method
type RegistryListParams struct {
	Refs   []*repo.DatasetRef
	Limit  int
	Offset int
}

// List returns the list of datasets that have been published to a registry
func (r *RegistryRequests) List(params *RegistryListParams, done *bool) error {
	if r.cli != nil {
		return r.cli.Call("RegistryRequests.List", params, done)
	}

	dsRefs, err := actions.RegistryList(r.node, params.Limit, params.Offset)
	if err != nil {
		return err
	}
	params.Refs = dsRefs
	return nil
}

// GetDataset returns a dataset that has been published to the registry
func (r *RegistryRequests) GetDataset(ref *repo.DatasetRef, res *repo.DatasetRef) error {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Get", ref, res)
	}

	if err := actions.RegistryDataset(r.node, ref); err != nil {
		return err
	}

	*res = *ref
	return nil
}
