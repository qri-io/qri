package lib

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/repo"
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

// FetchParams encapsulates parameters for a fetch request
type FetchParams struct {
	Ref        string
	RemoteName string
}

// Fetch pulls a logbook from a remote
func (r *RemoteMethods) Fetch(p *FetchParams, res *[]dsref.VersionInfo) error {
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

	addr, err := remote.Address(r.inst.Config(), p.RemoteName)
	if err != nil {
		return err
	}

	// TODO (b5) - need contexts yo
	ctx := context.TODO()
	logs, err := r.inst.RemoteClient().FetchLogs(ctx, reporef.ConvertToDsref(ref), addr)
	if err != nil {
		return err
	}

	// TODO (b5) - FetchLogs currently returns oplogs arranged in user > dataset > branch
	// hierarchy, and we need to descend to the branch oplog to get commit history
	// info. It might be nicer if FetchLogs instead returned the branch oplog, but
	// with .Parent() fields loaded & connected
	if len(logs.Logs) > 0 {
		logs = logs.Logs[0]
		if len(logs.Logs) > 0 {
			logs = logs.Logs[0]
		}
	}

	versions := logbook.Versions(logs, reporef.ConvertToDsref(ref), 0, -1)
	log.Debugf("found %d versions: %v", len(versions), versions)
	if len(versions) == 0 {
		return repo.ErrNoHistory
	}

	for i, v := range versions {
		local, hasErr := r.inst.Repo().Store().Has(ctx, v.Path)
		if hasErr != nil {
			continue
		}
		versions[i].Foreign = !local
	}

	*res = versions
	return nil
}

// PublicationParams encapsulates parmeters for dataset publication
type PublicationParams struct {
	Ref        string
	RemoteName string
	// All indicates all versions of a dataset and the dataset namespace should
	// be either published or removed
	All bool
}

// Publish posts a dataset version to a remote
func (r *RemoteMethods) Publish(p *PublicationParams, res *dsref.Ref) error {
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
	if pushLogsErr := r.inst.RemoteClient().PushLogs(ctx, reporef.ConvertToDsref(ref), addr); pushLogsErr != nil {
		log.Errorf("pushing logs: %s", pushLogsErr)
	}

	if err = r.inst.RemoteClient().PushDataset(ctx, ref, addr); err != nil {
		return err
	}

	ref.Published = true
	if err = base.SetPublishStatus(r.inst.node.Repo, &ref, ref.Published); err != nil {
		return err
	}

	*res = reporef.ConvertToDsref(ref)
	return nil
}

// Unpublish asks a remote to remove a dataset
func (r *RemoteMethods) Unpublish(p *PublicationParams, res *dsref.Ref) error {
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
	if removeLogsErr := r.inst.RemoteClient().RemoveLogs(ctx, reporef.ConvertToDsref(ref), addr); removeLogsErr != nil {
		log.Errorf("removing logs: %s", removeLogsErr.Error())
	}

	if err := r.inst.RemoteClient().RemoveDataset(ctx, ref, addr); err != nil {
		return err
	}

	ref.Published = false
	if err = base.SetPublishStatus(r.inst.node.Repo, &ref, ref.Published); err != nil {
		return err
	}

	*res = reporef.ConvertToDsref(ref)
	return nil
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
