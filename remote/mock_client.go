package remote

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/dsfs"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	repotest "github.com/qri-io/qri/repo/test"
)

// ErrNotImplemented is returned for methods that are not implemented
var ErrNotImplemented = fmt.Errorf("not implemented")

// OtherPeer represents another peer which the MockClient connects to
type OtherPeer struct {
	info     *cfgtest.PeerInfo
	repoRoot *repotest.TempRepo
	book     *logbook.Book
	resolver map[string]string
	dscache  map[string]string
}

// MockClient is a remote client suitable for tests
type MockClient struct {
	node *p2p.QriNode
	book *logbook.Book

	otherPeers map[string]*OtherPeer

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

	cli := &MockClient{
		node:       node,
		book:       book,
		otherPeers: map[string]*OtherPeer{},
		doneCh:     make(chan struct{}),
		shutdown:   cancel,
	}

	go func() {
		<-ctx.Done()
		// TODO (b5) - return an error here if client is in the process of pulling anything
		cli.doneErr = ctx.Err()
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
	return "", r.c.createTheirDataset(ctx, ref)
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

	// Create the dataset on the foreign side.
	if err := c.createTheirDataset(ctx, ref); err != nil {
		return nil, err
	}

	// Get the logs for that dataset, merge them into our own.
	if err := c.pullLogs(ctx, *ref, remoteAddr); err != nil {
		return nil, err
	}

	// Put the dataset into our repo as well
	vi, err := c.mockDagSync(ctx, *ref)
	if err != nil {
		return nil, err
	}

	return dsfs.LoadDataset(ctx, c.node.Repo.Filesystem(), vi.Path)
}

func (c *MockClient) createTheirDataset(ctx context.Context, ref *dsref.Ref) error {
	other := c.otherPeer(ref.Username)

	// Check if the dataset already exists
	if initID, exists := other.resolver[ref.Human()]; exists {
		if dsPath, ok := other.dscache[initID]; ok {
			ref.InitID = initID
			ref.ProfileID = other.info.EncodedPeerID
			ref.Path = dsPath
			return nil
		}
	}

	// TODO(dlong): HACK: This mockClient adds dataset to the *local* IPFS repo, instead
	// of the *foreign* IPFS repo. This is because there's no easy way to copy blocks
	// from one repo to another in tests. For now, this behavior works okay for our
	// existing tests, but will break if we need a test the expects different blocks to
	// exist on our repo versus theirs. The pull command is still doing useful work,
	// since the mockClient is producing different logbook and refstore info on each peer.
	// To fix this, create the dataset in the other.repoRoot.Repo.store instead, and
	// then down in mockDagSync copy the IPFS blocks from that store to the local store.
	fs := c.node.Repo.Filesystem()

	// Construct a simple dataset
	ds := dataset.Dataset{
		Commit: &dataset.Commit{},
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaObject,
		},
		BodyBytes: []byte("{}"),
	}
	err := ds.OpenBodyFile(ctx, fs)
	if err != nil {
		return err
	}

	// Allocate an initID for this dataset
	ref.InitID, err = other.book.WriteDatasetInit(ctx, ref.Name)
	if err != nil {
		return err
	}

	// Store with dsfs
	sw := dsfs.SaveSwitches{}
	path, err := dsfs.CreateDataset(ctx, fs, fs.DefaultWriteFS(), event.NilBus, &ds, nil, other.info.PrivKey, sw)
	if err != nil {
		return err
	}

	// Save the IPFS path with our fake refstore
	other.resolver[ref.Human()] = ref.InitID
	other.dscache[ref.InitID] = path
	ref.ProfileID = other.info.EncodedPeerID
	ref.Path = path

	// Add a save operation to logbook
	err = other.book.WriteVersionSave(ctx, ref.InitID, &ds, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *MockClient) otherPeer(username string) *OtherPeer {
	other, ok := c.otherPeers[username]
	if !ok {
		// Get test peer info, skipping 0th peer because many tests already use that one
		i := len(c.otherPeers) + 1
		info := cfgtest.GetTestPeerInfo(i)
		// Construct a tempRepo to hold IPFS data (not used, see HACK note above).
		tempRepo, err := repotest.NewTempRepoFixedProfileID(username, "")
		if err != nil {
			panic(err)
		}
		// Construct logbook
		fs, err := qfs.NewMemFilesystem(context.Background(), nil)
		if err != nil {
			panic(err)
		}
		book, err := logbook.NewJournal(info.PrivKey, username, event.NilBus, fs, "logbook.qfb")
		if err != nil {
			panic(err)
		}
		// Other peer represents a peer with the given username
		other = &OtherPeer{
			resolver: map[string]string{},
			dscache:  map[string]string{},
			info:     info,
			repoRoot: &tempRepo,
			book:     book,
		}
		c.otherPeers[username] = other
	}
	return other
}

// pullLogs creates a log from a temp logbook, and merges those into the
// client's logbook
func (c *MockClient) pullLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	log.Debug("MockClient.pullLogs")

	// Get other peer to retrieve its logbook
	other := c.otherPeer(ref.Username)
	theirBook := other.book
	theirLog, err := theirBook.UserDatasetBranchesLog(ctx, ref.InitID)
	if err != nil {
		return err
	}

	// Merge their logbook into ours
	if err = theirBook.SignLog(theirLog); err != nil {
		return err
	}
	return c.book.MergeLog(ctx, theirBook.Author(), theirLog)
}

// mockDagSync immitates a dagsync, pulling a dataset from a peer, and saving it with our refs
func (c *MockClient) mockDagSync(ctx context.Context, ref dsref.Ref) (*dsref.VersionInfo, error) {
	other := c.otherPeer(ref.Username)

	// Resolve the ref using the other peer's information
	initID := other.resolver[ref.Human()]
	dsPath := other.dscache[initID]

	// TODO(dustmop): HACK: Because we created the dataset to our own IPFS repo, there's no
	// need to copy the blocks. We should instead have added them to other.repoRoot, and here
	// copy the blocks to our own repo.

	// Add to our repository
	vi := dsref.VersionInfo{
		Path:      dsPath,
		ProfileID: ref.ProfileID,
		Username:  ref.Username,
		Name:      ref.Name,
	}
	if err := repo.PutVersionInfoShim(ctx, c.node.Repo, &vi); err != nil {
		return nil, err
	}

	return &vi, nil
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

// DoneErr gives an error that occurred during the shutdown process
func (c *MockClient) DoneErr() error {
	return c.doneErr
}

// Shutdown allows you to close the client before the parent context closes
func (c *MockClient) Shutdown() <-chan struct{} {
	c.shutdown()
	return c.Done()
}
