package lib

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/remote"
	reporef "github.com/qri-io/qri/repo/ref"
)

const allowedDagInfoSize uint64 = 10 * 1024 * 1024

// RemoteMethods encapsulates business logic of remote operation
// TODO (b5): switch to using an Instance instead of separate fields
type RemoteMethods struct {
	inst *Instance
}

// NewRemoteMethods creates a RemoteMethods pointer from either a node or an rpc.Client
func NewRemoteMethods(inst *Instance) *RemoteMethods {
	return &RemoteMethods{
		inst: inst,
	}
}

// CoreRequestsName implements the Requests interface
func (*RemoteMethods) CoreRequestsName() string { return "remote" }

// Feeds returns a listing of datasets from a number of feeds like featured and
// popular. Each feed is keyed by string in the response
func (r *RemoteMethods) Feeds(remoteName *string, res *map[string][]dsref.VersionInfo) error {
	if r.inst.rpc != nil {
		return checkRPCError(r.inst.rpc.Call("RemoteMethods.Feeds", remoteName, res))
	}
	ctx := context.TODO()

	addr, err := remote.Address(r.inst.Config(), *remoteName)
	if err != nil {
		return err
	}

	feed, err := r.inst.RemoteClient().Feeds(ctx, addr)
	if err != nil {
		return err
	}

	*res = feed
	return nil
}

// PreviewParams provides arguments to the preview method
type PreviewParams struct {
	RemoteName string
	Ref        string
}

// Preview requests a dataset preview from a remote
func (r *RemoteMethods) Preview(p *PreviewParams, res *dataset.Dataset) error {
	if r.inst.rpc != nil {
		return checkRPCError(r.inst.rpc.Call("RemoteMethods.Preview", p, res))
	}
	ctx := context.TODO()

	ref, err := dsref.Parse(p.Ref)
	if err != nil {
		return err
	}

	addr, err := remote.Address(r.inst.Config(), p.RemoteName)
	if err != nil {
		return err
	}

	pre, err := r.inst.RemoteClient().PreviewDatasetVersion(ctx, ref, addr)
	if err != nil {
		return err
	}

	*res = *pre
	return nil
}

// PushParams encapsulates parmeters for dataset publication
type PushParams struct {
	Ref        string
	RemoteName string
	// All indicates all versions of a dataset and the dataset namespace should
	// be either published or removed
	All bool
}

// Push posts a dataset version to a remote
func (r *RemoteMethods) Push(p *PushParams, res *dsref.Ref) error {
	if r.inst.rpc != nil {
		return checkRPCError(r.inst.rpc.Call("RemoteMethods.Push", p, res))
	}

	// TODO (b5) - need contexts yo
	ctx := context.TODO()

	ref, _, err := r.inst.ParseAndResolveRef(ctx, p.Ref, "local")
	if err != nil {
		return err
	}

	addr, err := remote.Address(r.inst.Config(), p.RemoteName)
	if err != nil {
		return err
	}

	if err = r.inst.RemoteClient().PushDataset(ctx, ref, addr); err != nil {
		return err
	}

	datasetRef := reporef.RefFromDsref(ref)
	datasetRef.Published = true
	if err = base.SetPublishStatus(ctx, r.inst.node.Repo, ref, true); err != nil {
		return err
	}

	*res = ref
	return nil
}

// Pull fetches a dataset version & logbook data from a remote
func (r *RemoteMethods) Pull(p *PushParams, res *dataset.Dataset) error {
	if r.inst.rpc != nil {
		return checkRPCError(r.inst.rpc.Call("RemoteMethods.Pull", p, res))
	}

	ref, err := dsref.Parse(p.Ref)
	if err != nil {
		return err
	}

	// TODO (b5) - need contexts yo
	ctx := context.TODO()

	ds, err := r.inst.RemoteClient().PullDataset(ctx, &ref, p.RemoteName)
	*res = *ds
	return err
}

// Remove asks a remote to remove a dataset
func (r *RemoteMethods) Remove(p *PushParams, res *dsref.Ref) error {
	if r.inst.rpc != nil {
		return checkRPCError(r.inst.rpc.Call("RemoteMethods.Remove", p, res))
	}

	ref, err := dsref.ParseHumanFriendly(p.Ref)
	if err != nil {
		if err == dsref.ErrNotHumanFriendly {
			return fmt.Errorf("can only remove entire dataset. run remove without a path")
		}
		return err
	}

	// TODO (b5) - need contexts yo
	ctx := context.TODO()

	if _, err := r.inst.ResolveReference(ctx, &ref, "local"); err != nil {
		return err
	}

	addr, err := remote.Address(r.inst.Config(), p.RemoteName)
	if err != nil {
		return err
	}

	if err := r.inst.RemoteClient().RemoveDataset(ctx, ref, addr); err != nil {
		return err
	}

	if err = base.SetPublishStatus(ctx, r.inst.node.Repo, ref, false); err != nil {
		return err
	}

	*res = ref
	return nil
}
