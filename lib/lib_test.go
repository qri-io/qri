package lib

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/access"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/repo"
	repotest "github.com/qri-io/qri/repo/test"
)

// base64-encoded Test Private Key, decoded in init
// peerId: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
var (
	testPk  = []byte(`CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`)
	privKey crypto.PrivKey

	testPeerProfile = &profile.Profile{
		Peername: "peer",
		ID:       profile.IDB58DecodeOrEmpty("QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"),
	}
)

func init() {
	data, err := base64.StdEncoding.DecodeString(string(testPk))
	if err != nil {
		panic(err)
	}
	testPk = data

	privKey, err = crypto.UnmarshalPrivateKey(testPk)
	if err != nil {
		panic(fmt.Errorf("error unmarshaling private key: %s", err.Error()))
	}
	testPeerProfile.PrivKey = privKey
}

func TestNewInstance(t *testing.T) {

	if _, err := NewInstance(context.Background(), ""); err == nil {
		t.Error("expected NewInstance to error when provided no repo path")
	}

	tr, err := repotest.NewTempRepo("foo", "new_instance_test", repotest.NewTestCrypto())
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Delete()

	cfg := config.DefaultConfigForTesting()
	cfg.Filesystems = []qfs.Config{
		{Type: "mem"},
		{Type: "local"},
	}
	cfg.Repo.Type = "mem"

	var firedEventWg sync.WaitGroup
	firedEventWg.Add(1)
	handler := func(_ context.Context, e event.Event) error {
		if e.Type == event.ETInstanceConstructed {
			firedEventWg.Done()
		}
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	got, err := NewInstance(ctx, tr.QriPath, OptConfig(cfg), OptEventHandler(handler, event.ETInstanceConstructed))
	if err != nil {
		t.Fatal(err)
	}

	expect := &Instance{
		cfg: cfg,
	}

	if err = CompareInstances(got, expect); err != nil {
		t.Error(err)
	}

	firedEventWg.Wait()

	finished := make(chan struct{})
	go func() {
		select {
		case <-time.NewTimer(time.Millisecond * 100).C:
			t.Errorf("done didn't fire within 100ms of canceling instance context")
		case <-got.Shutdown():
		}
		finished <- struct{}{}
	}()
	cancel()
	<-finished
}

func TestNewDefaultInstance(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir, err := ioutil.TempDir("", "TestNewDefaultInstance")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	qriPath := filepath.Join(tempDir, "qri")
	ipfsPath := filepath.Join(qriPath, "ipfs")
	t.Logf(tempDir)

	testCrypto := repotest.NewTestCrypto()
	if err = testCrypto.GenerateEmptyIpfsRepo(ipfsPath, ""); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfigForTesting()
	cfg.Filesystems = []qfs.Config{
		{Type: "ipfs", Config: map[string]interface{}{"path": ipfsPath}},
		{Type: "local"},
		{Type: "http"},
	}
	cfg.WriteToFile(filepath.Join(qriPath, "config.yaml"))

	_, err = NewInstance(ctx, qriPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(qriPath, "stats")); os.IsNotExist(err) {
		t.Errorf("NewInstance error: stats cache never created")
	}
}

func CompareInstances(a, b *Instance) error {
	if !reflect.DeepEqual(a.cfg, b.cfg) {
		return fmt.Errorf("config mismatch")
	}

	// TODO (b5): compare all instance fields
	return nil
}

func TestReceivers(t *testing.T) {
	ctx := context.Background()
	fs, err := muxfs.New(ctx, []qfs.Config{
		{Type: "mem"},
	})
	if err != nil {
		t.Fatal(err)
	}
	r, err := repo.NewMemRepoWithProfile(ctx, &profile.Profile{PrivKey: privKey}, fs, event.NilBus)
	if err != nil {
		t.Errorf("error creating mem repo: %s", err)
		return
	}
	n, err := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode).New(r)
	if err != nil {
		t.Errorf("error creating qri node: %s", err)
		return
	}

	node := n.(*p2p.QriNode)
	cfg := config.DefaultConfigForTesting()
	inst := &Instance{node: node, cfg: cfg}

	reqs := Receivers(inst)
	expect := 11
	if len(reqs) != expect {
		t.Errorf("unexpected number of receivers returned. expected: %d. got: %d\nhave you added/removed a receiver?", expect, len(reqs))
		return
	}
}

// pulled from base packages
// TODO - we should probably get a test package going at github.com/qri-io/qri/test
func addCitiesDataset(t *testing.T, node *p2p.QriNode) dsref.Ref {
	t.Helper()
	ctx := context.Background()
	tc, err := dstest.NewTestCaseFromDir(repotest.TestdataPath("cities"))
	if err != nil {
		t.Fatal(err.Error())
	}
	ds := tc.Input
	ds.Name = tc.Name
	ds.BodyBytes = tc.Body

	ref, err := saveDataset(ctx, node.Repo, ds, base.SaveSwitches{Pin: true})
	if err != nil {
		t.Fatal(err.Error())
	}
	return ref
}

func saveDataset(ctx context.Context, r repo.Repo, ds *dataset.Dataset, sw base.SaveSwitches) (dsref.Ref, error) {
	pro, err := r.Profile()
	if err != nil {
		return dsref.Ref{}, err
	}
	ref, _, err := base.PrepareSaveRef(ctx, pro, r.Logbook(), r.Logbook(), fmt.Sprintf("%s/%s", pro.Peername, ds.Name), "", false)
	if err != nil {
		return dsref.Ref{}, err
	}
	res, err := base.SaveDataset(ctx, r, r.Filesystem().DefaultWriteFS(), ref.InitID, ref.Path, ds, sw)
	if err != nil {
		return dsref.Ref{}, err
	}
	return dsref.ConvertDatasetToVersionInfo(res).SimpleRef(), nil
}

func TestInstanceEventSubscription(t *testing.T) {
	timeoutMs := time.Millisecond * 250
	if runtime.GOOS == "windows" {
		// TODO(dustmop): Why is windows slow? Perhaps its due to the IPFS lock.
		timeoutMs = time.Millisecond * 2500
	}
	// TODO (b5) - can't use testrunner for this test because event busses aren't
	// wired up correctly in the test runner constructor. The proper fix is to have
	// testrunner build it's instance using NewInstance
	ctx, done := context.WithTimeout(context.Background(), timeoutMs)
	defer done()

	tmpDir, err := ioutil.TempDir("", "event_sub_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfigForTesting()
	cfg.Repo.Type = "mem"
	// remove default ipfs fs, not needed for this test
	cfg.Filesystems = []qfs.Config{{Type: "mem"}}

	eventCh := make(chan event.Type, 1)
	expect := event.ETP2PGoneOnline
	eventHandler := func(_ context.Context, e event.Event) error {
		eventCh <- e.Type
		return nil
	}

	inst, err := NewInstance(ctx, tmpDir,
		OptConfig(cfg),
		// remove logbook, not needed for this test
		OptLogbook(&logbook.Book{}),
		OptEventHandler(eventHandler, expect),
	)
	if err != nil {
		t.Fatal(err)
	}

	go inst.Node().GoOnline(ctx)
	select {
	case evt := <-eventCh:
		if evt != expect {
			t.Errorf("received event mismatch. expected: %q, got: %q", expect, evt)
		}
	case <-ctx.Done():
		t.Error("expected event before context deadline, got none")
	}
}

func TestNewInstanceWithAccessControlPolicy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir, err := ioutil.TempDir("", "TestNewInstanceWithAccessControlPolicy")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	qriPath := filepath.Join(tempDir, "qri")
	ipfsPath := filepath.Join(qriPath, "ipfs")
	t.Logf(tempDir)

	testCrypto := repotest.NewTestCrypto()
	if err = testCrypto.GenerateEmptyIpfsRepo(ipfsPath, ""); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfigForTesting()
	cfg.Filesystems = []qfs.Config{
		{Type: "ipfs", Config: map[string]interface{}{"path": ipfsPath}},
		{Type: "local"},
		{Type: "http"},
	}

	cfg.Remote = &config.Remote{
		Enabled: true,
	}

	proID := profile.ID("foo")
	acpPath := filepath.Join(qriPath, access.DefaultAccessControlPolicyFilename)
	acp := []byte(`
	[
		{
			"title": "pull foo:bar dataset",
			"effect": "allow",
			"subject": "` + proID.String() + `",
			"resources": [
				"dataset:foo:bar"
			],
			"actions": [
				"remote:pull"
			]
		}
	]
`)

	if err := ioutil.WriteFile(acpPath, acp, 0644); err != nil {
		t.Fatalf("error writing access control policy: %s", err)
	}

	got, err := NewInstance(
		ctx,
		qriPath,
		OptConfig(cfg),
		OptRemoteOptions(
			[]remote.OptionsFunc{
				remote.OptLoadPolicyFileIfExists(acpPath),
			}),
	)
	if err != nil {
		t.Fatal(err)
	}

	p := got.remote.Policy()
	if p == nil {
		t.Fatal("expected remote policy to exist, got nil")
	}

	if err != nil {
		t.Fatal("error creating profileID:", err)
	}
	if err := p.Enforce(&profile.Profile{ID: proID, Peername: "foo"}, "dataset:foo:bar", "remote:pull"); err != nil {
		t.Errorf("expected no policy enforce error, got: %s", err)
	}
}
