package mock

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/repo"
	repotest "github.com/qri-io/qri/repo/test"
)

// ErrNotImplemented is returned for methods that are not implemented
var ErrNotImplemented = fmt.Errorf("not implemented")

// OtherPeer represents another peer which the Client connects to
type OtherPeer struct {
	keyData  *testkeys.KeyData
	repoRoot *repotest.TempRepo
	book     *logbook.Book
	resolver map[string]string
	dscache  map[string]string
}

// Client is a remote client suitable for tests
type Client struct {
	node *p2p.QriNode
	pub  event.Publisher

	otherPeers map[string]*OtherPeer

	doneCh   chan struct{}
	doneErr  error
	shutdown context.CancelFunc
}

var _ remote.Client = (*Client)(nil)

// NewClient returns a mock remote client. context passed to NewClient
// MUST use the `Shutdown` method or cancel externally for proper cleanup
func NewClient(ctx context.Context, node *p2p.QriNode, pub event.Publisher) (c remote.Client, err error) {
	ctx, cancel := context.WithCancel(ctx)

	cli := &Client{
		pub:        pub,
		node:       node,
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
func (c *Client) Feeds(ctx context.Context, remoteAddr string) (map[string][]dsref.VersionInfo, error) {
	return nil, ErrNotImplemented
}

// Feed is not implemented
func (c *Client) Feed(ctx context.Context, remoteAddr, feedName string, page, pageSize int) ([]dsref.VersionInfo, error) {
	return nil, ErrNotImplemented
}

// PreviewDatasetVersion is not implemented
func (c *Client) PreviewDatasetVersion(ctx context.Context, ref dsref.Ref, remoteAddr string) (*dataset.Dataset, error) {
	return nil, ErrNotImplemented
}

// FetchLogs is not implemented
func (c *Client) FetchLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) (*oplog.Log, error) {
	return nil, ErrNotImplemented
}

// NewRemoteRefResolver mocks a ref resolver off a foreign logbook
func (c *Client) NewRemoteRefResolver(addr string) dsref.Resolver {
	// TODO(b5) - switch based on address input? it would make for a better mock
	return &writeOnResolver{c: c}
}

// writeOnResolver creates dataset histories on the fly when
// ResolveRef is called, storing them for future
type writeOnResolver struct {
	c *Client
}

func (r *writeOnResolver) ResolveRef(ctx context.Context, ref *dsref.Ref) (string, error) {
	return "", r.c.createTheirDataset(ctx, ref)
}

// PushDataset is not implemented
func (c *Client) PushDataset(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// RemoveDataset is not implemented
func (c *Client) RemoveDataset(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// RemoveDatasetVersion is not implemented
func (c *Client) RemoveDatasetVersion(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// PullDataset adds a reference to a dataset using test peer info
func (c *Client) PullDataset(ctx context.Context, ref *dsref.Ref, remoteAddr string) (*dataset.Dataset, error) {
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

	ds, err := dsfs.LoadDataset(ctx, c.node.Repo.Filesystem(), vi.Path)
	if err != nil {
		return nil, err
	}

	info := dsref.ConvertDatasetToVersionInfo(ds)
	info.InitID = ref.InitID
	info.Username = ref.Username
	info.Name = ref.Name
	info.ProfileID = ref.ProfileID
	if err := c.pub.Publish(ctx, event.ETDatasetPulled, info); err != nil {
		return nil, err
	}
	return ds, err
}

func (c *Client) createTheirDataset(ctx context.Context, ref *dsref.Ref) error {
	other := c.otherPeer(ref.Username)

	// Check if the dataset already exists
	if initID, exists := other.resolver[ref.Human()]; exists {
		if dsPath, ok := other.dscache[initID]; ok {
			ref.InitID = initID
			ref.ProfileID = other.keyData.EncodedPeerID
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
	ref.InitID, err = other.book.WriteDatasetInit(ctx, other.book.Owner(), ref.Name)
	if err != nil {
		return err
	}

	// Store with dsfs
	sw := dsfs.SaveSwitches{}
	path, err := dsfs.CreateDataset(ctx, fs, fs.DefaultWriteFS(), event.NilBus, &ds, nil, other.keyData.PrivKey, sw)
	if err != nil {
		return err
	}

	// Save the IPFS path with our fake refstore
	other.resolver[ref.Human()] = ref.InitID
	other.dscache[ref.InitID] = path
	ref.ProfileID = other.keyData.EncodedPeerID
	ref.Path = path

	// Add a save operation to logbook
	ds.ID = ref.InitID
	err = other.book.WriteVersionSave(ctx, other.book.Owner(), &ds, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) otherPeer(username string) *OtherPeer {
	other, ok := c.otherPeers[username]
	if !ok {
		// Get test peer info, skipping 0th peer because many tests already use that one
		i := len(c.otherPeers) + 1
		kd := testkeys.GetKeyData(i)
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
		pro, err := profile.NewSparsePKProfile(username, kd.PrivKey)
		if err != nil {
			panic(err)
		}
		book, err := logbook.NewJournal(*pro, event.NilBus, fs, "logbook.qfb")
		if err != nil {
			panic(err)
		}
		// Other peer represents a peer with the given username
		other = &OtherPeer{
			resolver: map[string]string{},
			dscache:  map[string]string{},
			keyData:  kd,
			repoRoot: &tempRepo,
			book:     book,
		}
		c.otherPeers[username] = other
	}
	return other
}

// pullLogs creates a log from a temp logbook, and merges those into the
// client's logbook
func (c *Client) pullLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	// Get other peer to retrieve its logbook
	other := c.otherPeer(ref.Username)
	theirBook := other.book
	theirLog, err := theirBook.UserDatasetBranchesLog(ctx, ref.InitID)
	if err != nil {
		return err
	}

	// Merge their logbook into ours
	if err = theirLog.Sign(theirBook.Owner().PrivKey); err != nil {
		return err
	}
	return c.node.Repo.Logbook().MergeLog(ctx, theirBook.Owner().PrivKey.GetPublic(), theirLog)
}

// mockDagSync immitates a dagsync, pulling a dataset from a peer, and saving it with our refs
func (c *Client) mockDagSync(ctx context.Context, ref dsref.Ref) (*dsref.VersionInfo, error) {
	other := c.otherPeer(ref.Username)

	// Resolve the ref using the other peer's information
	initID := other.resolver[ref.Human()]
	dsPath := other.dscache[initID]

	// TODO(dustmop): HACK: Because we created the dataset to our own IPFS repo, there's no
	// need to copy the blocks. We should instead have added them to other.repoRoot, and here
	// copy the blocks to our own repo.

	// Add to our repository
	vi := dsref.VersionInfo{
		InitID:    initID,
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
func (c *Client) PushLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// PullDatasetVersion is not implemented
func (c *Client) PullDatasetVersion(ctx context.Context, ref *dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// RemoveLogs is not implemented
func (c *Client) RemoveLogs(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	return ErrNotImplemented
}

// Done returns a channel that the client will send on when finished closing
func (c *Client) Done() <-chan struct{} {
	return c.doneCh
}

// DoneErr gives an error that occurred during the shutdown process
func (c *Client) DoneErr() error {
	return c.doneErr
}

// Shutdown allows you to close the client before the parent context closes
func (c *Client) Shutdown() <-chan struct{} {
	c.shutdown()
	return c.Done()
}
