package remote

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	core "github.com/ipfs/go-ipfs/core"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func TestDatasetPullPushDeleteHTTP(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	hooksCalled := []string{}
	callCheck := func(s string) Hook {
		return func(ctx context.Context, pid profile.ID, ref repo.DatasetRef) error {
			hooksCalled = append(hooksCalled, s)
			return nil
		}
	}

	requireLogAndRefCallCheck := func(t *testing.T, s string) Hook {
		return func(ctx context.Context, pid profile.ID, ref repo.DatasetRef) error {
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
	}

	aCfg := &config.Remote{
		Enabled:       true,
		AllowRemoves:  true,
		AcceptSizeMax: 10000,
	}

	rem, err := NewRemote(tr.NodeA, aCfg, opts)
	if err != nil {
		t.Error(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/remote/refs", rem.RefsHTTPHandler())
	mux.Handle("/remote/logsync", rem.LogsyncHTTPHandler())
	mux.Handle("/remote/dsync", rem.DsyncHTTPHandler())
	server := httptest.NewServer(mux)

	worldBankRef := writeWorldBankPopulation(tr.Ctx, t, tr.NodeA.Repo)

	cli, err := NewClient(tr.NodeB)
	if err != nil {
		t.Error(err)
	}

	relRef := &repo.DatasetRef{Peername: worldBankRef.Peername, Name: worldBankRef.Name}
	if err := cli.ResolveHeadRef(tr.Ctx, relRef, server.URL); err != nil {
		t.Error(err)
	}

	if !relRef.Equal(worldBankRef) {
		t.Errorf("resolve mismatch. expected:\n%s\ngot:\n%s", worldBankRef, relRef)
	}

	if _, err := cli.FetchLogs(tr.Ctx, repo.ConvertToDsref(*relRef), server.URL); err != nil {
		t.Error(err)
	}
	if err := cli.PullDataset(tr.Ctx, &worldBankRef, server.URL); err != nil {
		t.Error(err)
	}

	videoViewRef := writeVideoViewStats(tr.Ctx, t, tr.NodeB.Repo)

	if err := cli.PushLogs(tr.Ctx, repo.ConvertToDsref(videoViewRef), server.URL); err != nil {
		t.Error(err)
	}
	if err := cli.PushDataset(tr.Ctx, videoViewRef, server.URL); err != nil {
		t.Error(err)
	}

	if err := cli.RemoveLogs(tr.Ctx, repo.ConvertToDsref(videoViewRef), server.URL); err != nil {
		t.Error(err)
	}
	if err := cli.RemoveDataset(tr.Ctx, videoViewRef, server.URL); err != nil {
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

type testRunner struct {
	Ctx          context.Context
	NodeA, NodeB *p2p.QriNode
}

func newTestRunner(t *testing.T) (tr *testRunner, cleanup func()) {
	var err error
	tr = &testRunner{
		Ctx: context.Background(),
	}

	nodes, _, err := p2ptest.MakeIPFSSwarm(tr.Ctx, true, 2)
	if err != nil {
		t.Fatal(err)
	}

	tr.NodeA = qriNode(t, "A", nodes[0], cfgtest.GetTestPeerInfo(0))
	tr.NodeB = qriNode(t, "B", nodes[1], cfgtest.GetTestPeerInfo(1))

	cleanup = func() {}
	return tr, cleanup
}

func qriNode(t *testing.T, peername string, node *core.IpfsNode, pi *cfgtest.PeerInfo) *p2p.QriNode {
	repo, err := p2ptest.MakeRepoFromIPFSNode(node, peername)
	if err != nil {
		t.Fatal(err)
	}

	qriNode, err := p2p.NewQriNode(repo, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err)
	}

	return qriNode
}

func writeWorldBankPopulation(ctx context.Context, t *testing.T, r repo.Repo) repo.DatasetRef {
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

	ref, err := base.CreateDataset(ctx, r, ioes.NewDiscardIOStreams(), ds, nil, false, true, false, true)
	if err != nil {
		t.Fatal(err)
	}

	return ref
}

func writeVideoViewStats(ctx context.Context, t *testing.T, r repo.Repo) repo.DatasetRef {
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

	ref, err := base.CreateDataset(ctx, r, ioes.NewDiscardIOStreams(), ds, nil, false, true, false, true)
	if err != nil {
		t.Fatal(err)
	}

	return ref
}
