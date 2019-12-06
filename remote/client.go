package remote

import (
	"context"

	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
)

// Client connects to remotes to perform synchronization tasks
type Client interface {
	ListDatasets(ctx context.Context, ds *repo.DatasetRef, term string, offset, limit int) ([]repo.DatasetRef, error)

	PushDataset(ctx context.Context, ref repo.DatasetRef, remoteAddr string) error
	PullDataset(ctx context.Context, ref *repo.DatasetRef, remoteAddr string) error
	RemoveDataset(ctx context.Context, ref repo.DatasetRef, remoteAddr string) error
	AddDataset(ctx context.Context, ref *repo.DatasetRef, remoteAddr string) error

	PushLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error
	PullLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error
	RemoveLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error

	ResolveHeadRef(ctx context.Context, ref *repo.DatasetRef, remoteAddr string) error
}
