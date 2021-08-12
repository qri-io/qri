package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"
	core "github.com/ipfs/go-ipfs/core"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/config"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/remote/access"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

func TestDatasetPullPushDeleteFeedsPreviewHTTP(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	hooksCalled := []string{}
	callCheck := func(s string) Hook {
		return func(ctx context.Context, pid profile.ID, ref dsref.Ref) error {
			hooksCalled = append(hooksCalled, s)
			return nil
		}
	}

	requireLogAndRefCallCheck := func(t *testing.T, s string) Hook {
		return func(ctx context.Context, pid profile.ID, ref dsref.Ref) error {
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

	wbp := writeWorldBankPopulation(tr.Ctx, t, tr.NodeA.Repo)
	cli := tr.NodeBClient(t)

	var (
		firedEvents            []event.Type
		pushProgressEventFired bool
		pullProgressEventFired bool
	)
	tr.NodeB.Repo.Bus().SubscribeTypes(func(_ context.Context, e event.Event) error {
		switch e.Type {
		case event.ETRemoteClientPushVersionProgress:
			pushProgressEventFired = true
		case event.ETRemoteClientPullVersionProgress:
			pullProgressEventFired = true
		default:
			firedEvents = append(firedEvents, e.Type)
		}
		return nil
	},
		event.ETRemoteClientPushVersionProgress,
		event.ETRemoteClientPullVersionProgress,
		event.ETRemoteClientPushVersionCompleted,
		event.ETRemoteClientPushDatasetCompleted,
		event.ETRemoteClientPullDatasetCompleted,
		event.ETRemoteClientRemoveDatasetCompleted,
	)

	progBuf := &bytes.Buffer{}

	relRef := &dsref.Ref{Username: wbp.Username, Name: wbp.Name}
	if _, err := cli.NewRemoteRefResolver(server.URL).ResolveRef(tr.Ctx, relRef); err != nil {
		t.Error(err)
	}

	if !relRef.Equals(wbp) {
		t.Errorf("resolve mismatch. expected:\n%s\ngot:\n%s", wbp, relRef)
	}

	if _, err := cli.PullDataset(tr.Ctx, &wbp, server.URL); err != nil {
		t.Error(err)
	}

	videoViewRef := writeVideoViewStats(tr.Ctx, t, tr.NodeB.Repo)

	if err := cli.PushDataset(tr.Ctx, videoViewRef, server.URL); err != nil {
		t.Error(err)
	}

	if err := cli.RemoveDataset(tr.Ctx, videoViewRef, server.URL); err != nil {
		t.Error(err)
	}

	if _, err := cli.Feeds(tr.Ctx, server.URL); err != nil {
		t.Error(err)
	}
	if _, err := cli.PreviewDatasetVersion(tr.Ctx, wbp, server.URL); err != nil {
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

	expectEventsOrder := []event.Type{
		event.ETRemoteClientPullDatasetCompleted,
		event.ETRemoteClientPushVersionCompleted,
		event.ETRemoteClientPushDatasetCompleted,
		event.ETRemoteClientRemoveDatasetCompleted,
	}

	if diff := cmp.Diff(expectEventsOrder, firedEvents); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}

	if !pushProgressEventFired {
		t.Error("expected push progress event to have fired at least once")
	}
	if !pullProgressEventFired {
		t.Error("expected pull progress event to have fired at least once")
	}

	if len(progBuf.String()) == 0 {
		// This is only a warning, mainly b/c some operating systems (linux) run
		// so quickly progress isn't written to the buffer
		t.Logf("warning: expected progress to be written to an output buffer, buf is empty")
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
	wbpRepoRef := reporef.RefFromDsref(wbp)
	setRefPublished(tr.Ctx, t, tr.NodeA.Repo, tr.NodeA.Repo.Profiles().Owner(), &wbpRepoRef)

	vvs := writeVideoViewStats(tr.Ctx, t, tr.NodeA.Repo)
	vvsRepoRef := reporef.RefFromDsref(vvs)
	setRefPublished(tr.Ctx, t, tr.NodeA.Repo, tr.NodeA.Repo.Profiles().Owner(), &vvsRepoRef)

	aCfg := &config.RemoteServer{
		Enabled:       true,
		AllowRemoves:  true,
		AcceptSizeMax: 10000,
	}

	rem, err := NewServer(tr.NodeA, aCfg, tr.NodeA.Repo.Logbook(), tr.NodeA.Repo.Bus())
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
				Username:      "A",
				Name:          "video_view_stats",
				Path:          "/ipfs/QmYUPbXCS3akX9sot95oVTV5jDCFuTHwG8wPcYw2rXwRvS",
				ProfileID:     "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Published:     true,
				MetaTitle:     "Video View Stats",
				BodySize:      4,
				BodyRows:      1,
				BodyFormat:    "json",
				CommitTime:    time.Time{},
				CommitTitle:   "initial commit",
				CommitMessage: "created dataset",
			},
			{
				Username:      "A",
				Name:          "world_bank_population",
				Path:          "/ipfs/QmcBiMNF7giCmbEjaAd5tSz7NxMXCBnU9NwCwyuprJgXZ6",
				ProfileID:     "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
				Published:     true,
				MetaTitle:     "World Bank Population",
				BodySize:      5,
				BodyRows:      1,
				BodyFormat:    "json",
				CommitTime:    time.Time{},
				CommitTitle:   "initial commit",
				CommitMessage: "created dataset",
			},
		},
	}

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("feed mismatch. (-want +got): \n%s", diff)
	}

}

func TestAccess(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	// add datasets to both nodes
	aRef := writeWorldBankPopulation(tr.Ctx, t, tr.NodeA.Repo)
	bRef := writeVideoViewStats(tr.Ctx, t, tr.NodeB.Repo)
	// establish nodeB as the client
	cli := tr.NodeBClient(t)
	// empty policy should allow no actions
	rem := tr.NodeARemote(t, OptPolicy(&access.Policy{}))
	server := tr.RemoteTestServer(rem)
	defer server.Close()

	if err := cli.PushDataset(tr.Ctx, bRef, server.URL); err.Error() != access.ErrAccessDenied.Error() {
		t.Errorf("expected %q when trying to push dataset to a remote that does not allow pushes, got %q instead", access.ErrAccessDenied, err)
	}

	allowPushPolicy := &access.Policy{}
	mustJSON(`
	[
		{
			"title": "allow subject to push its own datasets",
			"effect": "allow",
			"subject": "*",
			"resources": [
				"dataset:_subject:*"
			],
			"actions": [
				"remote:push"
			]
		}
	]
`, allowPushPolicy)
	rem.policy = allowPushPolicy

	if err := cli.PushDataset(tr.Ctx, bRef, server.URL); err != nil {
		t.Errorf("unexpected error when trying to push dataset to a remote that allows pushes: %q", err)
	}

	if err := cli.RemoveDataset(tr.Ctx, bRef, server.URL); err.Error() != access.ErrAccessDenied.Error() {
		t.Errorf("expected %q when trying to remove dataset from a remote that does not allow removes, got %q instead", access.ErrAccessDenied, err)
	}

	allowPushRemovePolicy := &access.Policy{}
	mustJSON(`
	[
		{
			"title": "allow subject to push and remove its own datasets",
			"effect": "allow",
			"subject": "*",
			"resources": [
				"dataset:_subject:*"
			],
			"actions": [
				"remote:push",
				"remote:remove"
			]
		}
	]
`, allowPushRemovePolicy)
	rem.policy = allowPushRemovePolicy

	if err := cli.RemoveDataset(tr.Ctx, bRef, server.URL); err != nil {
		t.Errorf("unexpected error when trying to remove a dataset from a remote that allows removes: %q", err)
	}

	if _, err := cli.PullDataset(tr.Ctx, &aRef, server.URL); err.Error() != access.ErrAccessDenied.Error() {
		t.Errorf("expected %q when trying to pull a dataset from a remote that does not allow pulls, got %q instead", access.ErrAccessDenied, err)
	}

	allowPullOwnPolicy := &access.Policy{}
	mustJSON(`
	[
		{
			"title": "allow subjects to pull datasets that are their own",
			"effect": "allow",
			"subject": "*",
			"resources": [
				"dataset:_subject:*"
			],
			"actions": [
				"remote:pull"
			]
		}
	]
`, allowPullOwnPolicy)
	rem.policy = allowPullOwnPolicy

	if _, err := cli.PullDataset(tr.Ctx, &aRef, server.URL); err.Error() != access.ErrAccessDenied.Error() {
		t.Errorf("expected %q when trying to pull a dataset from a remote that does not allow pulls, got %q instead", access.ErrAccessDenied, err)
	}

	allowAllPullsPolicy := &access.Policy{}
	mustJSON(`
	[
		{
			"title": "allow pulls of all datasets",
			"effect": "allow",
			"subject": "*",
			"resources": [
				"dataset:*"
			],
			"actions": [
				"remote:pull"
			]
		}
	]
	`, allowAllPullsPolicy)
	rem.policy = allowAllPullsPolicy

	if _, err := cli.PullDataset(tr.Ctx, &aRef, server.URL); err != nil {
		t.Errorf("unexpected error when trying to pull a dataset from a remote that allows all pulls: %q", err)
	}
}

type testRunner struct {
	Ctx          context.Context
	NodeA, NodeB *p2p.QriNode
}

func newTestRunner(t *testing.T) (tr *testRunner, cleanup func()) {
	var err error
	ctx, cancel := context.WithCancel(context.Background())
	tr = &testRunner{
		Ctx: ctx,
	}
	prevTs := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Time{} }

	nodes, _, err := p2ptest.MakeIPFSSwarm(tr.Ctx, true, 2)
	if err != nil {
		t.Fatal(err)
	}

	tr.NodeA = qriNode(ctx, t, tr, "A", nodes[0])
	tr.NodeB = qriNode(ctx, t, tr, "B", nodes[1])

	cleanup = func() {
		dsfs.Timestamp = prevTs
		cancel()
	}
	return tr, cleanup
}

func (tr *testRunner) NodeARemote(t *testing.T, opts ...OptionsFunc) *Server {
	aCfg := &config.RemoteServer{
		Enabled:       true,
		AllowRemoves:  true,
		AcceptSizeMax: 10000,
	}

	rem, err := NewServer(tr.NodeA, aCfg, tr.NodeA.Repo.Logbook(), tr.NodeA.Repo.Bus(), opts...)
	if err != nil {
		t.Fatal(err)
	}
	return rem
}

func (tr *testRunner) RemoteTestServer(rem *Server) *httptest.Server {
	m := mux.NewRouter()
	rem.AddDefaultRoutes(m)
	return httptest.NewServer(m)
}

func (tr *testRunner) NodeBClient(t *testing.T) Client {
	cli, err := NewClient(tr.Ctx, tr.NodeB, tr.NodeB.Repo.Bus())
	if err != nil {
		t.Fatal(err)
	}
	return cli
}

func qriNode(ctx context.Context, t *testing.T, tr *testRunner, peername string, node *core.IpfsNode) *p2p.QriNode {
	repo, err := p2ptest.MakeRepoFromIPFSNode(tr.Ctx, node, peername, event.NewBus(ctx))
	if err != nil {
		t.Fatal(err)
	}

	localResolver := dsref.SequentialResolver(repo.Dscache(), repo)
	qriNode, err := p2p.NewQriNode(repo, testcfg.DefaultP2PForTesting(), repo.Bus(), localResolver)
	if err != nil {
		t.Fatal(err)
	}

	return qriNode
}

func writeWorldBankPopulation(ctx context.Context, t *testing.T, r repo.Repo) dsref.Ref {
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
	author := r.Logbook().Owner()
	return saveDataset(ctx, r, author, ds)
}

func setRefPublished(ctx context.Context, t *testing.T, r repo.Repo, author *profile.Profile, ref *reporef.DatasetRef) {
	if err := base.SetPublishStatus(ctx, r, author, reporef.ConvertToDsref(*ref), true); err != nil {
		t.Fatal(err)
	}
}

func writeVideoViewStats(ctx context.Context, t *testing.T, r repo.Repo) dsref.Ref {
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
	author := r.Logbook().Owner()
	return saveDataset(ctx, r, author, ds)
}

func saveDataset(ctx context.Context, r repo.Repo, author *profile.Profile, ds *dataset.Dataset) dsref.Ref {
	headRef := ""
	book := r.Logbook()
	initID, err := book.RefToInitID(dsref.Ref{Username: author.Peername, Name: ds.Name})
	if err == nil {
		got, _ := r.GetRef(reporef.DatasetRef{Peername: author.Peername, Name: ds.Name})
		headRef = got.Path
	} else if err == logbook.ErrNotFound {
		initID, err = book.WriteDatasetInit(ctx, author, ds.Name)
	}
	if err != nil {
		panic(err)
	}
	res, err := base.SaveDataset(ctx, r, r.Filesystem().DefaultWriteFS(), r.Profiles().Owner(), initID, headRef, ds, nil, base.SaveSwitches{})
	if err != nil {
		panic(err)
	}
	ref := dsref.ConvertDatasetToVersionInfo(res).SimpleRef()
	ref.InitID = initID
	return ref
}

func mustJSON(data string, v interface{}) {
	if err := json.Unmarshal([]byte(data), v); err != nil {
		panic(err)
	}
}
