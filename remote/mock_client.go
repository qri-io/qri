package remote

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/dsfs"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/identity"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// ErrNotImplemented is returned for methods that are not implemented
var ErrNotImplemented = fmt.Errorf("not implemented")

// MockClient is a remote client suitable for tests
type MockClient struct {
	node *p2p.QriNode
	book *logbook.Book

	storagePath  string
	foreignBooks map[string]*logbook.Book

	doneCh   chan struct{}
	doneErr  error
	shutdown context.CancelFunc
}

var _ Client = (*MockClient)(nil)

// NewMockClient returns a mock remote client. context passed to NewMockClient
// MUST use the `Shutdown` method or cancel externally for proper cleanup
func NewMockClient(ctx context.Context, node *p2p.QriNode, book *logbook.Book) (c Client, err error) {
	log.Debug("creating mock remote client")
	ctx, cancel := context.WithCancel(ctx)
	tmpDir, err := ioutil.TempDir("", "qri-mock-remote-client")

	cli := &MockClient{
		node:         node,
		book:         book,
		storagePath:  tmpDir,
		foreignBooks: map[string]*logbook.Book{},
		doneCh:       make(chan struct{}),
		close:        cancel,
	}

	go func() {
		<-ctx.Done()
		// TODO (b5) - return an error here if client is in the process of pulling anything
		cli.doneErr = ctx.Err()
		os.RemoveAll(cli.storagePath)
		close(cli.doneCh)
	}()

	return cli, nil
}

// Feeds is not implemented
func (c *MockClient) Feeds(ctx context.Context, remoteAddr string) (map[string][]dsref.VersionInfo, error) {
	return nil, ErrNotImplemented
}

// Feed is not implemented
func (c *MockClient) Feed(ctx context.Context, remoteAddr, feedName string, page, pageSize int) ([]dsref.VersionInfo, error) {
	return nil, ErrNotImplemented
}

// PreviewDatasetVersion is not implemented
func (c *MockClient) PreviewDatasetVersion(ctx context.Context, ref dsref.Ref, remoteAddr string) (*dataset.Dataset, error) {
	return nil, ErrNotImplemented
}

// FetchLogs is not implemented
func (c *MockClient) FetchLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) (*oplog.Log, error) {
	return nil, ErrNotImplemented
}

// NewRemoteRefResolver mocks a ref resolver off a foreign logbook
func (c *MockClient) NewRemoteRefResolver(addr string) dsref.Resolver {
	// TODO(b5) - switch based on address input? it would make for a better mock
	return &writeOnResolver{c: c}
}

// writeOnResolver creates dataset histories on the fly when
// ResolveRef is called, storing them for future
type writeOnResolver struct {
	c *MockClient
}

func (r *writeOnResolver) ResolveRef(ctx context.Context, ref *dsref.Ref) (string, error) {
	log.Debugf("MockClient writeOnResolver.ResolveRef ref=%q", ref)
	_, err := r.c.makeRefExist(ctx, ref)
	return "", err
}

func (c *MockClient) makeRefExist(ctx context.Context, ref *dsref.Ref) (*logbook.Book, error) {
	book, ok := c.foreignBooks[ref.Name]
	if !ok {
		fs, err := qfs.NewMemFilesystem(context.Background(), nil)
		if err != nil {
			return nil, err
		}
		// TODO (b5) - use unique keypairs for each peer
		otherPeerInfo := cfgtest.GetTestPeerInfo(1)
		book, err = logbook.NewJournal(otherPeerInfo.PrivKey, ref.Username, event.NilBus, fs, "logbook.qfb")
		if err != nil {
			return nil, err
		}
		c.foreignBooks[ref.Username] = book
	}

	if _, resolveErr := book.ResolveRef(ctx, ref); resolveErr == nil {
		return book, nil
	}

	var err error
	ref.InitID, err = book.WriteDatasetInit(ctx, ref.Name)
	if err != nil {
		return nil, err
	}

	zeroTime := time.Time{}
	ref.Path = "QmExample"
	err = book.WriteVersionSave(ctx, ref.InitID, &dataset.Dataset{
		Commit: &dataset.Commit{
			Timestamp: zeroTime.In(time.UTC),
			Title:     "their commit",
		},
		Path: ref.Path,
	})
	if err != nil {
		return nil, err
	}

	ref.ProfileID, err = identity.KeyIDFromPub(book.AuthorPubKey())
	return book, err
}

// PushDataset is not implemented
func (c *MockClient) PushDataset(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// RemoveDataset is not implemented
func (c *MockClient) RemoveDataset(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// RemoveDatasetVersion is not implemented
func (c *MockClient) RemoveDatasetVersion(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// PullDataset adds a reference to a dataset using test peer info
func (c *MockClient) PullDataset(ctx context.Context, ref *dsref.Ref, remoteAddr string) (*dataset.Dataset, error) {
	log.Debugf("MockClient.PullDataset ref=%q", ref)

	if err := c.pullLogs(ctx, *ref, remoteAddr); err != nil {
		return nil, err
	}

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
		return nil, err
	}

	vi := &dsref.VersionInfo{
		Path:      path,
		ProfileID: info.PeerID.Pretty(),
		Username:  ref.Username,
		Name:      ref.Name,
	}

	// Store ref for a mock dataset.
	if err := repo.PutVersionInfoShim(c.node.Repo, vi); err != nil {
		// if err := c.node.Repo.PutRef(*ref); err != nil {
		return nil, err
	}

	return dsfs.LoadDataset(ctx, c.node.Repo.Store(), path)
}

// pullLogs creates a log from a temp logbook, and merges those into the
// client's logbook
func (c *MockClient) pullLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	log.Debug("MockClient.pullLogs")
	book, err := c.makeRefExist(ctx, &ref)
	if err != nil {
		return err
	}

	l, err := book.UserDatasetBranchesLog(ctx, ref.InitID)
	if err != nil {
		return err
	}

	if err = book.SignLog(l); err != nil {
		return err
	}

	return c.book.MergeLog(ctx, book.Author(), l)
}

// PushLogs is not implemented
func (c *MockClient) PushLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// PullDatasetVersion is not implemented
func (c *MockClient) PullDatasetVersion(ctx context.Context, ref *dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// RemoveLogs is not implemented
func (c *MockClient) RemoveLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// Done returns a channel that the client will send on when finished closing
func (c *MockClient) Done() <-chan struct{} {
	return c.doneCh
}

// DoneErr gives an error that occured during the shutdown process
func (c *MockClient) DoneErr() error {
	return c.doneErr
}

// Shutdown allows you to close the client before the parent context closes
func (c *MockClient) Shutdown() <-chan struct{} {
	c.shutdown()
	return c.Done()
}
