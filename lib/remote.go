package lib

import (
	"context"
	"fmt"
	"net/http"

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
func (r *RemoteMethods) Feeds(ctx context.Context, p *EmptyParams) (map[string][]dsref.VersionInfo, error) {
	if r.inst.http != nil {
		res := map[string][]dsref.VersionInfo{}
		err := r.inst.http.Call(ctx, AEFeeds, p, &res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	location := ""
	// TODO(dustmop): Once scope is in use
	//location = scope.SourceName()

	addr, err := remote.Address(r.inst.GetConfig(), location)
	if err != nil {
		return nil, err
	}

	feed, err := r.inst.RemoteClient().Feeds(ctx, addr)
	if err != nil {
		return nil, err
	}
	return feed, nil
}

// PreviewParams provides arguments to the preview method
type PreviewParams struct {
	Ref string
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *PreviewParams) UnmarshalFromRequest(r *http.Request) error {
	if p == nil {
		p = &PreviewParams{}
	}
	if p.Ref == "" {
		p.Ref = r.FormValue("refstr")
	}
	return nil
}

// Preview requests a dataset preview from a remote
func (r *RemoteMethods) Preview(ctx context.Context, p *PreviewParams) (*dataset.Dataset, error) {
	if r.inst.http != nil {
		res := &dataset.Dataset{}
		err := r.inst.http.Call(ctx, AEPreview, p, res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	ref, err := dsref.Parse(p.Ref)
	if err != nil {
		return nil, err
	}

	// TODO(dustmop): When `scope` is in use, get the source from it
	source := ""
	addr, err := remote.Address(r.inst.GetConfig(), source)
	if err != nil {
		return nil, err
	}

	res, err := r.inst.RemoteClient().PreviewDatasetVersion(ctx, ref, addr)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// PushParams encapsulates parmeters for dataset publication
type PushParams struct {
	Ref string `schema:"ref" json:"refstr"`
	// All indicates all versions of a dataset and the dataset namespace should
	// be either published or removed
	All bool
}

// Push posts a dataset version to a remote
func (r *RemoteMethods) Push(ctx context.Context, p *PushParams) (*dsref.Ref, error) {
	if r.inst.http != nil {
		res := &dsref.Ref{}
		err := r.inst.http.Call(ctx, AEPush, p, res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	ref, _, err := r.inst.ParseAndResolveRef(ctx, p.Ref, "local")
	if err != nil {
		return nil, err
	}

	// TODO(dustmop): When `scope` is in use, get the source from it
	source := ""
	addr, err := remote.Address(r.inst.GetConfig(), source)
	if err != nil {
		return nil, err
	}

	if err = r.inst.RemoteClient().PushDataset(ctx, ref, addr); err != nil {
		return nil, err
	}

	datasetRef := reporef.RefFromDsref(ref)
	datasetRef.Published = true
	if err = base.SetPublishStatus(ctx, r.inst.node.Repo, ref, true); err != nil {
		return nil, err
	}

	return &ref, nil
}

// Remove asks a remote to remove a dataset
func (r *RemoteMethods) Remove(ctx context.Context, p *PushParams) (*dsref.Ref, error) {
	if r.inst.http != nil {
		res := &dsref.Ref{}
		qvars := map[string]string{
			"refstr": p.Ref,
			"remote": "network",
		}
		err := r.inst.http.CallMethod(ctx, AEPush, http.MethodDelete, qvars, res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	ref, err := dsref.ParseHumanFriendly(p.Ref)
	if err != nil {
		if err == dsref.ErrNotHumanFriendly {
			return nil, fmt.Errorf("can only remove entire dataset. run remove without a path")
		}
		return nil, err
	}

	if _, err := r.inst.ResolveReference(ctx, &ref, "local"); err != nil {
		return nil, err
	}

	// TODO(dustmop): When `scope` is in use, get the source from it
	source := ""
	addr, err := remote.Address(r.inst.GetConfig(), source)
	if err != nil {
		return nil, err
	}

	if err := r.inst.RemoteClient().RemoveDataset(ctx, ref, addr); err != nil {
		return nil, err
	}

	if err = base.SetPublishStatus(ctx, r.inst.node.Repo, ref, false); err != nil {
		return nil, err
	}

	return &ref, nil
}
