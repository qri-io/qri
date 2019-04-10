package lib

import (
	"net/rpc"

	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// RegistryMethods work with a qri registry
type RegistryMethods interface {
	Methods
	Publish(ref *repo.DatasetRef, done *bool) error
	Unpublish(ref *repo.DatasetRef, done *bool) error
	Pin(ref *repo.DatasetRef, done *bool) error
	Unpin(ref *repo.DatasetRef, done *bool) error
	List(params *RegistryListParams, done *bool) error
	GetDataset(ref *repo.DatasetRef, res *repo.DatasetRef) error
}

// NewRegistryMethods creates a registryMethods pointer from either a repo
// or an rpc.Client
func NewRegistryMethods(inst Instance) RegistryMethods {
	return registryMethods{
		node: inst.Node(),
		cli:  inst.RPC(),
	}
}

// registryMethods defines business logic for working with registries
type registryMethods struct {
	node *p2p.QriNode
	cli  *rpc.Client
}

// MethodsKind implements the Requests interface
func (registryMethods) MethodsKind() string { return "RegistryMethods" }

// Publish a dataset to a registry
func (r registryMethods) Publish(ref *repo.DatasetRef, done *bool) (err error) {
	if r.cli != nil {
		return r.cli.Call("RegistryMethods.Publish", ref, done)
	}
	return actions.Publish(r.node, *ref)
}

// Unpublish a dataset from a registry
func (r registryMethods) Unpublish(ref *repo.DatasetRef, done *bool) error {
	if r.cli != nil {
		return r.cli.Call("RegistryMethods.Unpublish", ref, done)
	}
	return actions.Unpublish(r.node, *ref)
}

// Pin asks a registry to host a copy of a dataset
func (r registryMethods) Pin(ref *repo.DatasetRef, done *bool) (err error) {
	if r.cli != nil {
		return r.cli.Call("RegistryMethods.Pin", ref, done)
	}
	return actions.Pin(r.node, *ref)
}

// Unpin reverses the pin process, asking a registry to stop hosting a copy of
// an already-pinned dataset
func (r registryMethods) Unpin(ref *repo.DatasetRef, done *bool) error {
	if r.cli != nil {
		return r.cli.Call("RegistryMethods.Unpin", ref, done)
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
func (r registryMethods) List(params *RegistryListParams, done *bool) error {
	if r.cli != nil {
		return r.cli.Call("RegistryMethods.List", params, done)
	}

	dsRefs, err := actions.RegistryList(r.node, params.Limit, params.Offset)
	if err != nil {
		return err
	}
	params.Refs = dsRefs
	return nil
}

// GetDataset returns a dataset that has been published to the registry
func (r registryMethods) GetDataset(ref *repo.DatasetRef, res *repo.DatasetRef) error {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Get", ref, res)
	}

	// Handle `qri use` to get the current default dataset
	if err := DefaultSelectedRef(r.node.Repo, ref); err != nil {
		return err
	}

	if err := actions.RegistryDataset(r.node, ref); err != nil {
		return err
	}

	*res = *ref
	return nil
}
