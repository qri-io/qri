package remote

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// ErrNotImplemented is returned for methods that are not implemented
var ErrNotImplemented = fmt.Errorf("not implemented")

// MockClient is a remote client suitable for tests
type MockClient struct {
	node *p2p.QriNode
}

// NewMockClient returns a mock remote client
func NewMockClient(node *p2p.QriNode) (c Client, err error) {
	return &MockClient{node: node}, nil
}

// ListDatasets is not implemented
func (c *MockClient) ListDatasets(ctx context.Context, ds *repo.DatasetRef, term string, offset, limit int) (res []repo.DatasetRef, err error) {
	return nil, ErrNotImplemented
}

// PushDataset is not implemented
func (c *MockClient) PushDataset(ctx context.Context, ref repo.DatasetRef, remoteAddr string) error {
	return ErrNotImplemented
}

// FetchLogs is not implemented
func (c *MockClient) FetchLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) (*oplog.Log, error) {
	return nil, ErrNotImplemented
}

// CloneLogs is not implemented
func (c *MockClient) CloneLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// RemoveDataset is not implemented
func (c *MockClient) RemoveDataset(ctx context.Context, ref repo.DatasetRef, remoteAddr string) error {
	return ErrNotImplemented
}

// AddDataset adds a reference to a dataset using test peer info
func (c *MockClient) AddDataset(ctx context.Context, ref *repo.DatasetRef, remoteAddr string) error {
	// Get a test peer, but skip the first peer (usually used for tests)
	info := cfgtest.GetTestPeerInfo(1)

	// Construct a simple dataset
	ds := dataset.Dataset{
		Commit: &dataset.Commit{},
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaObject,
		},
		BodyBytes: []byte("{}"),
	}
	_ = ds.OpenBodyFile(ctx, nil)

	// Store with dsfs
	path, err := dsfs.CreateDataset(ctx, c.node.Repo.Store(), &ds, nil, c.node.Repo.PrivateKey(), false, false, false)
	if err != nil {
		return err
	}

	// Fill in details for the reference
	ref.ProfileID = profile.ID(info.PeerID)
	ref.Path = path

	// Store ref for a mock dataset.
	if err := c.node.Repo.PutRef(*ref); err != nil {
		return err
	}
	return nil
}

// PushLogs is not implemented
func (c *MockClient) PushLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// PullDataset is not implemented
func (c *MockClient) PullDataset(ctx context.Context, ref *repo.DatasetRef, remoteAddr string) error {
	return ErrNotImplemented
}

// RemoveLogs is not implemented
func (c *MockClient) RemoveLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// ResolveHeadRef is not implemented
func (c *MockClient) ResolveHeadRef(ctx context.Context, ref *repo.DatasetRef, remoteAddr string) error {
	return ErrNotImplemented
}
