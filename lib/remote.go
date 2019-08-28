package lib

import (
	"context"

	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/repo"
)

const allowedDagInfoSize uint64 = 10 * 1024 * 1024

// RemoteMethods encapsulates business logic of remote operation
// TODO (b5): switch to using an Instance instead of separate fields
type RemoteMethods struct {
	inst *Instance
	cli  *remote.Client
}

// NewRemoteMethods creates a RemoteMethods pointer from either a node or an rpc.Client
func NewRemoteMethods(inst *Instance) *RemoteMethods {
	return &RemoteMethods{
		inst: inst,
		cli:  inst.remoteClient,
	}
}

// CoreRequestsName implements the Requests interface
func (*RemoteMethods) CoreRequestsName() string { return "remote" }

// PublicationParams encapsulates parmeters for dataset publication
type PublicationParams struct {
	Ref        string
	RemoteName string
	// All indicates all versions of a dataset amd the dataset namespace should
	// be either published or removed
	All bool
}

// Publish posts a dataset version to a remote
func (r *RemoteMethods) Publish(p *PublicationParams, res *repo.DatasetRef) error {
	if r.inst.rpc != nil {
		return r.inst.rpc.Call("DatasetRequests.Publish", p, res)
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(r.inst.Repo(), &ref); err != nil {
		return err
	}
	*res = ref

	addr, err := remote.Address(r.inst.Config(), p.RemoteName)
	if err != nil {
		return err
	}

	// TODO (b5) - need contexts yo
	ctx := context.TODO()

	if err = r.cli.PushDataset(ctx, ref, addr); err != nil {
		return err
	}

	res.Published = true
	return actions.SetPublishStatus(r.inst.node, res, res.Published)
}

// Unpublish asks a remote to remove a dataset
func (r *RemoteMethods) Unpublish(p *PublicationParams, res *repo.DatasetRef) error {
	if r.inst.rpc != nil {
		return r.inst.rpc.Call("DatasetRequests.Unpublish", p, res)
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(r.inst.Repo(), &ref); err != nil {
		return err
	}

	*res = ref

	addr, err := remote.Address(r.inst.Config(), p.RemoteName)
	if err != nil {
		return err
	}

	// TODO (b5) - need contexts yo
	ctx := context.TODO()

	if err := r.cli.RemoveDataset(ctx, ref, addr); err != nil {
		return err
	}

	res.Published = false
	return actions.SetPublishStatus(r.inst.node, res, res.Published)
}

// PullDataset fetches a dataset ref from a remote
func (r *RemoteMethods) PullDataset(p *PublicationParams, res *bool) error {
	if r.inst.rpc != nil {
		return r.inst.rpc.Call("DatasetRequests.Unpublish", p, res)
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}

	// TODO (b5) - need contexts yo
	ctx := context.TODO()

	err = r.cli.PullDataset(ctx, &ref, p.RemoteName)
	return err
}
