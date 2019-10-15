package lib

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/config"
	libtest "github.com/qri-io/qri/lib/test"
	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// base64-encoded Test Private Key, decoded in init
// peerId: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
var (
	testPk  = []byte(`CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`)
	privKey crypto.PrivKey

	testPeerProfile = &profile.Profile{
		Peername: "peer",
		ID:       "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
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

	// call LoadPlugins once with the empty string b/c we only rely on standard
	// plugin set
	if err := loadIPFSPluginsOnce(""); err != nil {
		panic(err)
	}
}

func TestNewInstance(t *testing.T) {
	var err error
	cfg := config.DefaultConfigForTesting()
	cfg.Store.Type = "map"
	cfg.Repo.Type = "mem"

	got, err := NewInstance(context.Background(), os.TempDir(), OptConfig(cfg))
	if err != nil {
		t.Error(err)
		return
	}

	expect := &Instance{
		cfg: cfg,
	}

	if err = CompareInstances(got, expect); err != nil {
		t.Error(err)
	}

	if _, err = NewInstance(context.Background(), ""); err == nil {
		t.Error("expected NewInstance to error when provided no repo path")
	}
}

func TestNewDefaultInstance(t *testing.T) {
	prevIPFSEnvLocation := os.Getenv("IPFS_PATH")
	prevDefaultIPFSLocation := defaultIPFSLocation

	os.Setenv("IPFS_PATH", "")

	tempDir, err := ioutil.TempDir(os.TempDir(), "TestNewDefaultInstance")
	if err != nil {
		t.Fatal(err)
	}
	defaultIPFSLocation = tempDir

	defer func() {
		defaultIPFSLocation = prevDefaultIPFSLocation
		os.Setenv("IPFS_PATH", prevIPFSEnvLocation)
		os.RemoveAll(tempDir)
	}()

	testCrypto := libtest.NewTestCrypto()
	if err = testCrypto.GenerateEmptyIpfsRepo(tempDir, ""); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfigForTesting()
	cfg.Store.Path = tempDir
	cfg.WriteToFile(filepath.Join(tempDir, "config.yaml"))

	_, err = NewInstance(context.Background(), tempDir)
	if err != nil {
		t.Fatal(err)
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
	store := cafs.NewMapstore()
	r, err := repo.NewMemRepo(&profile.Profile{PrivKey: privKey}, store, qfs.NewMemFS(), profile.NewMemStore())
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

func testdataPath(path string) string {
	return filepath.Join(os.Getenv("GOPATH"), "/src/github.com/qri-io/qri/repo/test/testdata", path)
}

// pulled from actions and base packages
// TODO - we should probably get a test package going at github.com/qri-io/qri/test
func addCitiesDataset(t *testing.T, node *p2p.QriNode) repo.DatasetRef {
	ctx := context.Background()
	tc, err := dstest.NewTestCaseFromDir(testdataPath("cities"))
	if err != nil {
		t.Fatal(err.Error())
	}
	ds := tc.Input
	ds.Name = tc.Name
	ds.BodyBytes = tc.Body

	ref, err := actions.SaveDataset(ctx, node, ds, nil, nil, actions.SaveDatasetSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Fatal(err.Error())
	}
	return ref
}

func addNowTransformDataset(t *testing.T, node *p2p.QriNode) repo.DatasetRef {
	ctx := context.Background()
	tc, err := dstest.NewTestCaseFromDir("testdata/now_tf")
	if err != nil {
		t.Fatal(err.Error())
	}
	ds := tc.Input
	ds.Name = tc.Name
	ds.Transform.ScriptPath = "testdata/now_tf/transform.star"

	ref, err := actions.SaveDataset(ctx, node, ds, nil, nil, actions.SaveDatasetSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Fatal(err.Error())
	}
	return ref
}
