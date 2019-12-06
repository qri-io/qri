package lib

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/repo"
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

// FetchParams encapsulates parameters for a fetch request
type FetchParams struct {
	Ref        string
	RemoteName string
}

// Fetch pulls a logbook from a remote
func (r *RemoteMethods) Fetch(p *FetchParams, res *repo.DatasetRef) error {
	if r.inst.rpc != nil {
		return r.inst.rpc.Call("RemoteMethods.Fetch", p, res)
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(r.inst.Repo(), &ref); err != nil {
		if err == repo.ErrNotFound {
			err = nil
		} else {
			return err
		}
	}
	*res = ref

	addr, err := remote.Address(r.inst.Config(), p.RemoteName)
	if err != nil {
		return err
	}

	// TODO (b5) - need contexts yo
	ctx := context.TODO()
	return r.inst.RemoteClient().PullLogs(ctx, repo.ConvertToDsref(ref), addr)
}

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
		return r.inst.rpc.Call("RemoteMethods.Publish", p, res)
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}
	if ref.Path != "" {
		return fmt.Errorf("can only publish entire dataset, cannot use version %s", ref.Path)
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

	// TODO (b5) - we're early in log syncronization days. This is going to fail a bunch
	// while we work to upgrade the stack. Long term we may want to consider a mechanism
	// for allowing partial completion where only one of logs or dataset pushing works
	// by doing both in parallel and reporting issues on both
	if pushLogsErr := r.inst.RemoteClient().PushLogs(ctx, repo.ConvertToDsref(ref), addr); pushLogsErr != nil {
		log.Errorf("pushing logs: %s", pushLogsErr)
	}

	if err = r.inst.RemoteClient().PushDataset(ctx, ref, addr); err != nil {
		return err
	}

	res.Published = true
	return base.SetPublishStatus(r.inst.node.Repo, res, res.Published)
}

// Unpublish asks a remote to remove a dataset
func (r *RemoteMethods) Unpublish(p *PublicationParams, res *repo.DatasetRef) error {
	if r.inst.rpc != nil {
		return r.inst.rpc.Call("RemoteMethods.Unpublish", p, res)
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}

	if ref.Path != "" {
		return fmt.Errorf("can only unpublish entire dataset, cannot use version %s", ref.Path)
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

	// TODO (b5) - we're early in log syncronization days. This is going to fail a bunch
	// while we work to upgrade the stack. Long term we may want to consider a mechanism
	// for allowing partial completion where only one of logs or dataset pushing works
	// by doing both in parallel and reporting issues on both
	if removeLogsErr := r.inst.RemoteClient().RemoveLogs(ctx, repo.ConvertToDsref(ref), addr); removeLogsErr != nil {
		log.Error("removing logs: %s", removeLogsErr.Error())
	}

	if err := r.inst.RemoteClient().RemoveDataset(ctx, ref, addr); err != nil {
		return err
	}

	res.Published = false
	return base.SetPublishStatus(r.inst.node.Repo, res, res.Published)
}

// PullDataset fetches a dataset ref from a remote
func (r *RemoteMethods) PullDataset(p *PublicationParams, res *bool) error {
	if r.inst.rpc != nil {
		return r.inst.rpc.Call("RemoteMethods.PullDataset", p, res)
	}

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}

	// TODO (b5) - need contexts yo
	ctx := context.TODO()

	err = r.inst.RemoteClient().PullDataset(ctx, &ref, p.RemoteName)
	return err
}
