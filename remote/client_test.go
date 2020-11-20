package remote

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/config"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/identity"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

func TestClientDone(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cli, err := NewClient(ctx, tr.NodeA, tr.NodeA.Repo.Bus())
	if err != nil {
		t.Fatal(err)
	}

	testDone := make(chan struct{})
	go func() {
		<-cli.Done()
		if doneErr := cli.DoneErr(); doneErr == nil {
			t.Errorf("expected a context cancellation error from client done, got nil")
		}
		testDone <- struct{}{}
	}()

	cancel()
	select {
	case <-testDone:
	case <-time.NewTimer(time.Millisecond * 100).C:
		t.Errorf("test didn't complete within 100 ms oc cancellation")
	}
}

func TestErrNoClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var client Client = (*client)(nil)

	if _, err := client.PullDataset(ctx, &dsref.Ref{}, ""); err != ErrNoRemoteClient {
		t.Errorf("error mismatch expected: %q, got: %q", ErrNoRemoteClient, err)
	}
	if err := client.RemoveDataset(ctx, dsref.Ref{}, ""); err != ErrNoRemoteClient {
		t.Errorf("error mismatch expected: %q, got: %q", ErrNoRemoteClient, err)
	}
	if _, err := client.PreviewDatasetVersion(ctx, dsref.Ref{}, ""); err != ErrNoRemoteClient {
		t.Errorf("error mismatch expected: %q, got: %q", ErrNoRemoteClient, err)
	}
	if err := client.RemoveDatasetVersion(ctx, dsref.Ref{}, ""); err != ErrNoRemoteClient {
		t.Errorf("error mismatch expected: %q, got: %q", ErrNoRemoteClient, err)
	}
	if _, err := client.FetchLogs(ctx, dsref.Ref{}, ""); err != ErrNoRemoteClient {
		t.Errorf("error mismatch expected: %q, got: %q", ErrNoRemoteClient, err)
	}
	if err := client.PushDataset(ctx, dsref.Ref{}, ""); err != ErrNoRemoteClient {
		t.Errorf("error mismatch expected: %q, got: %q", ErrNoRemoteClient, err)
	}
}

func TestNewRemoteRefResolver(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	remA := tr.NodeARemote(t)
	s := tr.RemoteTestServer(remA)
	cli := tr.NodeBClient(t)
	resolver := cli.NewRemoteRefResolver(s.URL)

	dsrefspec.AssertResolverSpec(t, resolver, func(r dsref.Ref, author identity.Author, log *oplog.Log) error {
		return remA.Node().Repo.Logbook().MergeLog(context.Background(), author, log)
	})
}

func TestClientFeedsAndPreviews(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	worldBankRef := writeWorldBankPopulation(tr.Ctx, t, tr.NodeA.Repo)
	wbp := reporef.RefFromDsref(worldBankRef)
	setRefPublished(t, tr.NodeA.Repo, &wbp)

	rem := tr.NodeARemote(t)
	server := tr.RemoteTestServer(rem)
	defer server.Close()

	cli := tr.NodeBClient(t)

	feeds, err := cli.Feeds(tr.Ctx, server.URL)
	if err != nil {
		t.Error(err)
	}

	expect := map[string][]dsref.VersionInfo{
		"recent": {
			{
				Username:   "A",
				Name:       "world_bank_population",
				Path:       "/ipfs/QmXEbqJUq4d1siXAiL4tXqfm1jYrQkziqx6LyoiKqqhnwh",
				MetaTitle:  "World Bank Population",
				BodySize:   5,
				BodyRows:   1,
				BodyFormat: "json",
			},
		},
	}

	if diff := cmp.Diff(expect, feeds); diff != "" {
		t.Errorf("feeds result mismatch (-want +got): \n%s", diff)
	}

	ds, err := cli.PreviewDatasetVersion(tr.Ctx, worldBankRef, server.URL)
	if err != nil {
		t.Error(err)
	}

	dstest.CompareGoldenDatasetAndUpdateIfEnvVarSet(t, "testdata/expect/TestClientFeedsAndPreviews.json", ds)
}

func newMemRepoTestNode(t *testing.T) *p2p.QriNode {
	ctx := context.Background()
	fs := qfs.NewMemFS()
	pi := cfgtest.GetTestPeerInfo(0)
	pro := &profile.Profile{
		Peername: "remote_test_peer",
		ID:       profile.IDFromPeerID(pi.PeerID),
		PrivKey:  pi.PrivKey,
	}
	mr, err := repo.NewMemRepo(ctx, pro, newTestFS(ctx, fs), event.NilBus)
	if err != nil {
		t.Fatal(err.Error())
	}
	localResolver := dsref.SequentialResolver(mr.Dscache(), mr)
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting(), event.NilBus, localResolver)
	if err != nil {
		t.Fatal(err.Error())
	}
	return node
}

func newTestFS(ctx context.Context, fs qfs.Filesystem) *muxfs.Mux {
	mux, err := muxfs.New(ctx, []qfs.Config{})
	if err != nil {
		panic(err)
	}
	if err := mux.SetFilesystem(fs); err != nil {
		panic(err)
	}
	return mux
}

// Convert from test nodes to non-test nodes.
// copied from p2p/peers_test.go
func asQriNodes(testPeers []p2ptest.TestablePeerNode) []*p2p.QriNode {
	// Convert from test nodes to non-test nodes.
	peers := make([]*p2p.QriNode, len(testPeers))
	for i, node := range testPeers {
		peers[i] = node.(*p2p.QriNode)
	}
	return peers
}

func connectMapStores(peers []*p2p.QriNode) {
	for i, s0 := range peers {
		for _, s1 := range peers[i+1:] {
			m0 := (s0.Repo.Filesystem().Filesystem(qfs.MemFilestoreType)).(*qfs.MemFS)
			m1 := (s1.Repo.Filesystem().Filesystem(qfs.MemFilestoreType)).(*qfs.MemFS)
			m0.AddConnection(m1)
		}
	}
}
