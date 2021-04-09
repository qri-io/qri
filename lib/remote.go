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
	d dispatcher
}

// Name returns the name of this method group
func (m RemoteMethods) Name() string {
	return "remote"
}

// Attributes defines attributes for each method
func (m RemoteMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"feeds":   {AEFeeds, "POST"},
		"preview": {AEPreview, "POST"},
		"push":    {AEPush, "POST"},
		"remove":  {AERemoteRemove, "POST"},
	}
}

// Feeds returns a listing of datasets from a number of feeds like featured and
// popular. Each feed is keyed by string in the response
func (m RemoteMethods) Feeds(ctx context.Context, p *EmptyParams) (map[string][]dsref.VersionInfo, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "feeds"), p)
	if res, ok := got.(map[string][]dsref.VersionInfo); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// PreviewParams provides arguments to the preview method
type PreviewParams struct {
	Ref string
}

// Preview requests a dataset preview from a remote
func (m RemoteMethods) Preview(ctx context.Context, p *PreviewParams) (*dataset.Dataset, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "preview"), p)
	if res, ok := got.(*dataset.Dataset); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// PushParams encapsulates parmeters for dataset publication
type PushParams struct {
	Ref    string `schema:"ref" json:"ref"`
	Remote string
	// All indicates all versions of a dataset and the dataset namespace should
	// be either published or removed
	All bool
}

// Push posts a dataset version to a remote
func (m RemoteMethods) Push(ctx context.Context, p *PushParams) (*dsref.Ref, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "push"), p)
	if res, ok := got.(*dsref.Ref); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Remove asks a remote to remove a dataset
func (m RemoteMethods) Remove(ctx context.Context, p *PushParams) (*dsref.Ref, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "remove"), p)
	if res, ok := got.(*dsref.Ref); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// remoteImpl holds the method implementations for RemoteMethods
type remoteImpl struct{}

// Feeds returns a listing of datasets from a number of feeds like featured and
// popular. Each feed is keyed by string in the response
func (remoteImpl) Feeds(scope scope, p *EmptyParams) (map[string][]dsref.VersionInfo, error) {
	addr, err := remote.Address(scope.Config(), scope.SourceName())
	if err != nil {
		return nil, err
	}

	feed, err := scope.RemoteClient().Feeds(scope.Context(), addr)
	if err != nil {
		return nil, err
	}
	return feed, nil
}

// Preview requests a dataset preview from a remote
func (remoteImpl) Preview(scope scope, p *PreviewParams) (*dataset.Dataset, error) {
	ref, err := dsref.Parse(p.Ref)
	if err != nil {
		return nil, err
	}

	addr, err := remote.Address(scope.Config(), scope.SourceName())
	if err != nil {
		return nil, err
	}

	res, err := scope.RemoteClient().PreviewDatasetVersion(scope.Context(), ref, addr)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Push posts a dataset version to a remote
func (remoteImpl) Push(scope scope, p *PushParams) (*dsref.Ref, error) {
	if scope.SourceName() != "local" {
		return nil, fmt.Errorf("can only push from local storage")
	}

	ref, location, err := scope.ParseAndResolveRef(scope.Context(), p.Ref)
	if err != nil {
		return nil, err
	}
/*
	addr, err := remote.Address(scope.Config(), p.Remote)
	if err != nil {
		return nil, err
	}
*/
	if err = scope.RemoteClient().PushDataset(scope.Context(), ref, location); err != nil {
		return nil, err
	}

	datasetRef := reporef.RefFromDsref(ref)
	datasetRef.Published = true
	if err = base.SetPublishStatus(scope.Context(), scope.Repo(), ref, true); err != nil {
		return nil, err
	}

	return &ref, nil
}

// Remove asks a remote to remove a dataset
func (remoteImpl) Remove(scope scope, p *PushParams) (*dsref.Ref, error) {
	ref, err := dsref.ParseHumanFriendly(p.Ref)
	if err != nil {
		if err == dsref.ErrNotHumanFriendly {
			return nil, fmt.Errorf("can only remove entire dataset. run remove without a path")
		}
		return nil, err
	}

	if _, err := scope.ResolveReference(scope.Context(), &ref); err != nil {
		return nil, err
	}

	addr, err := remote.Address(scope.Config(), p.Remote)
	if err != nil {
		return nil, err
	}

	if err := scope.RemoteClient().RemoveDataset(scope.Context(), ref, addr); err != nil {
		return nil, err
	}

	if err = base.SetPublishStatus(scope.Context(), scope.Repo(), ref, false); err != nil {
		return nil, err
	}

	return &ref, nil
}
