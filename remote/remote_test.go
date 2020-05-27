package remote

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	core "github.com/ipfs/go-ipfs/core"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/config"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

func TestDatasetPullPushDeleteFeedsPreviewHTTP(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	hooksCalled := []string{}
	callCheck := func(s string) Hook {
		return func(ctx context.Context, pid profile.ID, ref reporef.DatasetRef) error {
			hooksCalled = append(hooksCalled, s)
			return nil
		}
	}

	requireLogAndRefCallCheck := func(t *testing.T, s string) Hook {
		return func(ctx context.Context, pid profile.ID, ref reporef.DatasetRef) error {
			if ref.String() == "" {
				t.Errorf("hook %s expected reference to be populated", s)
			}
			if l, ok := OplogFromContext(ctx); !ok {
				t.Errorf("hook %s expected log to be in context. got: %v", s, l)
			}
			return callCheck(s)(ctx, pid, ref)
		}
	}

	opts := func(o *Options) {
		o.DatasetPushPreCheck = callCheck("DatasetPushPreCheck")
		o.DatasetPushFinalCheck = callCheck("DatasetPushFinalCheck")
		o.DatasetPushed = callCheck("DatasetPushed")
		o.DatasetPulled = callCheck("DatasetPulled")
		o.DatasetRemoved = callCheck("DatasetRemoved")

		o.LogPushPreCheck = callCheck("LogPushPreCheck")
		o.LogPushFinalCheck = requireLogAndRefCallCheck(t, "LogPushFinalCheck")
		o.LogPushed = requireLogAndRefCallCheck(t, "LogPushed")
		o.LogPullPreCheck = callCheck("LogPullPreCheck")
		o.LogPulled = callCheck("LogPulled")
		o.LogRemovePreCheck = callCheck("LogRemovePreCheck")
		o.LogRemoved = callCheck("LogRemoved")

		o.FeedPreCheck = callCheck("FeedPreCheck")
		o.PreviewPreCheck = callCheck("PreviewPreCheck")
	}

	rem := tr.NodeARemote(t, opts)
	server := tr.RemoteTestServer(rem)
	defer server.Close()

	worldBankRef := writeWorldBankPopulation(tr.Ctx, t, tr.NodeA.Repo)

	cli := tr.NodeBClient(t)

	relRef := &reporef.DatasetRef{Peername: worldBankRef.Peername, Name: worldBankRef.Name}
	if err := cli.ResolveHeadRef(tr.Ctx, relRef, server.URL); err != nil {
		t.Error(err)
	}

	if !relRef.Equal(worldBankRef) {
		t.Errorf("resolve mismatch. expected:\n%s\ngot:\n%s", worldBankRef, relRef)
	}

	if _, err := cli.FetchLogs(tr.Ctx, reporef.ConvertToDsref(*relRef), server.URL); err != nil {
		t.Error(err)
	}
	if err := cli.PullDataset(tr.Ctx, &worldBankRef, server.URL); err != nil {
		t.Error(err)
	}

	videoViewRef := writeVideoViewStats(tr.Ctx, t, tr.NodeB.Repo)

	if err := cli.PushLogs(tr.Ctx, reporef.ConvertToDsref(videoViewRef), server.URL); err != nil {
		t.Error(err)
	}
	if err := cli.PushDataset(tr.Ctx, videoViewRef, server.URL); err != nil {
		t.Error(err)
	}

	if err := cli.RemoveLogs(tr.Ctx, reporef.ConvertToDsref(videoViewRef), server.URL); err != nil {
		t.Error(err)
	}
	if err := cli.RemoveDataset(tr.Ctx, videoViewRef, server.URL); err != nil {
		t.Error(err)
	}

	if _, err := cli.Feeds(tr.Ctx, server.URL); err != nil {
		t.Error(err)
	}
	if _, err := cli.Preview(tr.Ctx, reporef.ConvertToDsref(worldBankRef), server.URL); err != nil {
		t.Error(err)
	}

	expectHooksCallOrder := []string{
		"LogPullPreCheck",
		"LogPulled",
		"DatasetPulled",
		"LogPushPreCheck",
		"LogPushFinalCheck",
		"LogPushed",
		"DatasetPushPreCheck",
		"DatasetPushFinalCheck",
		"DatasetPushed",
		"LogRemovePreCheck",
		"LogRemoved",
		"DatasetRemoved",
		"FeedPreCheck",
		"PreviewPreCheck",
	}

	if diff := cmp.Diff(expectHooksCallOrder, hooksCalled); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestAddress(t *testing.T) {
	if _, err := Address(&config.Config{}, ""); err == nil {
		t.Error("expected error, got nil")
	}

	cfg := &config.Config{
		Registry: &config.Registry{
			Location: "👋",
		},
	}

	addr, err := Address(cfg, "")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if cfg.Registry.Location != addr {
		t.Errorf("default location mismatch. expected: '%s', got: '%s'", cfg.Registry.Location, addr)
	}

	cfg.Remotes = &config.Remotes{"foo": "bar"}
	addr, err = Address(cfg, "foo")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if "bar" != addr {
		t.Errorf("default location mismatch. expected: '%s', got: '%s'", "bar", addr)
	}

	_, err = Address(cfg, "baz")
	if err == nil {
		t.Errorf("expected bad lookup to error")
	}
}

func TestFeeds(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	wbp := writeWorldBankPopulation(tr.Ctx, t, tr.NodeA.Repo)
	publishRef(t, tr.NodeA.Repo, &wbp)

	vvs := writeVideoViewStats(tr.Ctx, t, tr.NodeA.Repo)
	publishRef(t, tr.NodeA.Repo, &vvs)

	aCfg := &config.Remote{
		Enabled:       true,
		AllowRemoves:  true,
		AcceptSizeMax: 10000,
	}

	rem, err := NewRemote(tr.NodeA, aCfg)
	if err != nil {
		t.Fatal(err)
	}

	if rem.Feeds == nil {
		t.Errorf("expected RepoFeeds to be created by default. got nil")
	}

	got, err := rem.Feeds.Feeds(tr.Ctx, "")
	if err != nil {
		t.Error(err)
	}

	expect := map[string][]dsref.VersionInfo{
		"recent": {
			{
				Username:   "A",
				Name:       "video_view_stats",
				Path:       "/ipfs/QmPZ3W5291qJ9mq1fPpTmxfbDMc2ewXKiw2qGXSGBeQtWn",
				MetaTitle:  "Video View Stats",
				BodySize:   4,
				BodyRows:   1,
				BodyFormat: "json",
				CommitTime: time.Time{},
			},
			{
				Username:   "A",
				Name:       "world_bank_population",
				Path:       "/ipfs/Qmb5Qaigk9teHrWSyXf7UxnRH3L28BV6zV1cqaWsLn3z7p",
				MetaTitle:  "World Bank Population",
				BodySize:   5,
				BodyRows:   1,
				BodyFormat: "json",
				CommitTime: time.Time{},
			},
		},
	}

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("feed mismatch. (-want +got): \n%s", diff)
	}

}

type testRunner struct {
	Ctx          context.Context
	NodeA, NodeB *p2p.QriNode
}

func newTestRunner(t *testing.T) (tr *testRunner, cleanup func()) {
	var err error
	tr = &testRunner{
		Ctx: context.Background(),
	}
	prevTs := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Time{} }

	nodes, _, err := p2ptest.MakeIPFSSwarm(tr.Ctx, true, 2)
	if err != nil {
		t.Fatal(err)
	}

	tr.NodeA = qriNode(t, tr, "A", nodes[0], cfgtest.GetTestPeerInfo(0))
	tr.NodeB = qriNode(t, tr, "B", nodes[1], cfgtest.GetTestPeerInfo(1))

	cleanup = func() {
		dsfs.Timestamp = prevTs
	}
	return tr, cleanup
}

func (tr *testRunner) NodeARemote(t *testing.T, opts ...func(o *Options)) *Remote {
	aCfg := &config.Remote{
		Enabled:       true,
		AllowRemoves:  true,
		AcceptSizeMax: 10000,
	}

	rem, err := NewRemote(tr.NodeA, aCfg, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return rem
}

func (tr *testRunner) RemoteTestServer(rem *Remote) *httptest.Server {
	mux := http.NewServeMux()
	rem.AddDefaultRoutes(mux)
	return httptest.NewServer(mux)
}

func (tr *testRunner) NodeBClient(t *testing.T) Client {
	cli, err := NewClient(tr.NodeB)
	if err != nil {
		t.Fatal(err)
	}
	return cli
}

func qriNode(t *testing.T, tr *testRunner, peername string, node *core.IpfsNode, pi *cfgtest.PeerInfo) *p2p.QriNode {
	repo, err := p2ptest.MakeRepoFromIPFSNode(tr.Ctx, node, peername)
	if err != nil {
		t.Fatal(err)
	}

	qriNode, err := p2p.NewQriNode(repo, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err)
	}

	return qriNode
}

func writeWorldBankPopulation(ctx context.Context, t *testing.T, r repo.Repo) reporef.DatasetRef {
	ds := &dataset.Dataset{
		Name: "world_bank_population",
		Commit: &dataset.Commit{
			Title: "initial commit",
		},
		Meta: &dataset.Meta{
			Title: "World Bank Population",
		},
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[100]")))
	return saveDataset(ctx, r, "peer", ds)
}

func publishRef(t *testing.T, r repo.Repo, ref *reporef.DatasetRef) {
	if err := base.SetPublishStatus(r, ref, true); err != nil {
		t.Fatal(err)
	}
}

func writeVideoViewStats(ctx context.Context, t *testing.T, r repo.Repo) reporef.DatasetRef {
	ds := &dataset.Dataset{
		Name: "video_view_stats",
		Commit: &dataset.Commit{
			Title: "initial commit",
		},
		Meta: &dataset.Meta{
			Title: "Video View Stats",
		},
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[10]")))
	return saveDataset(ctx, r, "peer", ds)
}

func saveDataset(ctx context.Context, r repo.Repo, peername string, ds *dataset.Dataset) reporef.DatasetRef {
	sw := base.SaveSwitches{}
	headRef := ""
	book := r.Logbook()
	initID, err := book.RefToInitID(dsref.Ref{Username: peername, Name: ds.Name})
	if err == nil {
		got, _ := r.GetRef(reporef.DatasetRef{Peername: peername, Name: ds.Name})
		headRef = got.Path
	} else if err == logbook.ErrNotFound {
		initID, err = book.WriteDatasetInit(ctx, ds.Name)
	}
	if err != nil {
		panic(err)
	}
	datasetRef, err := base.SaveDataset(ctx, r, initID, headRef, ds, sw)
	if err != nil {
		panic(err)
	}
	return datasetRef
}
