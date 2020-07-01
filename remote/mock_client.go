package remote

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/dsfs"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/identity"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

// ErrNotImplemented is returned for methods that are not implemented
var ErrNotImplemented = fmt.Errorf("not implemented")

// MockClient is a remote client suitable for tests
type MockClient struct {
	node *p2p.QriNode
	book *logbook.Book
}

// NewMockClient returns a mock remote client
func NewMockClient(node *p2p.QriNode, book *logbook.Book) (c Client, err error) {
	return &MockClient{node: node, book: book}, nil
}

// ListDatasets is not implemented
func (c *MockClient) ListDatasets(ctx context.Context, ds *reporef.DatasetRef, term string, offset, limit int) (res []reporef.DatasetRef, err error) {
	return nil, ErrNotImplemented
}

// PushDataset is not implemented
func (c *MockClient) PushDataset(ctx context.Context, ref reporef.DatasetRef, remoteAddr string) error {
	return ErrNotImplemented
}

// FetchLogs is not implemented
func (c *MockClient) FetchLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) (*oplog.Log, error) {
	return nil, ErrNotImplemented
}

// CloneLogs creates a log from a temp logbook, and merges those into the client's logbook
func (c *MockClient) CloneLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	theirBookPath := filepath.Join(tmpdir, "their_logbook.qfs")

	otherUsername := ref.Username
	otherPeerInfo := cfgtest.GetTestPeerInfo(1)
	sender := identity.NewAuthor("abc", otherPeerInfo.PubKey)

	fs := qfs.NewMemFS()

	foreignBuilder := logbook.NewLogbookTempBuilder(nil, otherPeerInfo.PrivKey, otherUsername, fs, theirBookPath)

	initID := foreignBuilder.DatasetInit(ctx, nil, ref.Name)
	// NOTE: Need to assign ProfileID because nothing is resolving the username
	ref.ProfileID = otherPeerInfo.EncodedPeerID
	ref.Path = "QmExample"
	foreignBuilder.Commit(ctx, nil, initID, "their commit", ref.Path)
	foreignBook := foreignBuilder.Logbook()
	foreignLog, err := foreignBook.UserDatasetBranchesLog(ctx, initID)
	if err != nil {
		panic(err)
	}

	err = foreignBook.SignLog(foreignLog)
	if err != nil {
		panic(err)
	}

	return c.book.MergeLog(ctx, sender, foreignLog)
}

// RemoveDataset is not implemented
func (c *MockClient) RemoveDataset(ctx context.Context, ref reporef.DatasetRef, remoteAddr string) error {
	return ErrNotImplemented
}

// AddDataset adds a reference to a dataset using test peer info
func (c *MockClient) AddDataset(ctx context.Context, ref *reporef.DatasetRef, remoteAddr string) error {
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
	sw := dsfs.SaveSwitches{}
	path, err := dsfs.CreateDataset(ctx, c.node.Repo.Store(), c.node.Repo.Store(), &ds, nil, c.node.Repo.PrivateKey(), sw)
	if err != nil {
		return err
	}

	// Fill in details for the reference
	ref.ProfileID = profile.IDFromPeerID(info.PeerID)
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
func (c *MockClient) PullDataset(ctx context.Context, ref *reporef.DatasetRef, remoteAddr string) error {
	return ErrNotImplemented
}

// RemoveLogs is not implemented
func (c *MockClient) RemoveLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// ResolveHeadRef is not implemented
func (c *MockClient) ResolveHeadRef(ctx context.Context, ref *reporef.DatasetRef, remoteAddr string) error {
	return ErrNotImplemented
}

// NewRemoteRefResolver is not implemented
func (c *MockClient) NewRemoteRefResolver(addr string) dsref.Resolver {
	return nil
}

// Feeds is not implemented
func (c *MockClient) Feeds(ctx context.Context, remoteAddr string) (map[string][]dsref.VersionInfo, error) {
	return nil, ErrNotImplemented
}

// Preview is not implemented
func (c *MockClient) Preview(ctx context.Context, ref dsref.Ref, remoteAddr string) (*dataset.Dataset, error) {
	return nil, ErrNotImplemented
}
