package lib

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/httpfs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry/regclient"
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
}

// TODO (b5): restore
// func TestReceivers(t *testing.T) {
// 	store := cafs.NewMapstore()
// 	r, err := repo.NewMemRepo(&profile.Profile{}, store, qfs.NewMemFS(store), profile.NewMemStore(), nil)
// 	if err != nil {
// 		t.Errorf("error creating mem repo: %s", err)
// 		return
// 	}
// 	n, err := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode).New(r)
// 	if err != nil {
// 		t.Errorf("error creating qri node: %s", err)
// 		return
// 	}

// 	node := n.(*p2p.QriNode)
// 	reqs := Receivers(node, config.DefaultConfigForTesting(), "")
// 	if len(reqs) != 9 {
// 		t.Errorf("unexpected number of receivers returned. expected: %d. got: %d\nhave you added/removed a receiver?", 9, len(reqs))
// 		return
// 	}
// }

func newTestInstance(t *testing.T) Instance {
	node := newTestQriNode(t)
	cfg := config.DefaultConfigForTesting()
	ctx, teardown := context.WithCancel(context.Background())

	// TODO (b5): this is a hack until we can write configurations that
	// fully specify a test instance setup. Currently our setup isn't flexible
	// enough to capture all nuance necessary to create a test node, which is
	// a problem
	return &instance{
		ctx:      ctx,
		teardown: teardown,
		node:     node,
		cfg:      cfg,
	}
}

func newTestInstanceFromQriNode(node *p2p.QriNode) Instance {
	cfg := config.DefaultConfigForTesting()
	ctx, teardown := context.WithCancel(context.Background())
	// TODO (b5): this is a hack until we can write configurations that
	// fully specify a test instance setup. Currently our setup isn't flexible
	// enough to capture all nuance necessary to create a test node, which is
	// a problem
	return &instance{
		ctx:      ctx,
		teardown: teardown,
		node:     node,
		cfg:      cfg,
	}

}

func testdataPath(path string) string {
	return filepath.Join(os.Getenv("GOPATH"), "/src/github.com/qri-io/qri/repo/test/testdata", path)
}

func newTestQriNode(t *testing.T) *p2p.QriNode {
	ms := cafs.NewMapstore()
	r, err := repo.NewMemRepo(testPeerProfile, ms, newTestFS(ms), profile.NewMemStore(), nil)
	if err != nil {
		t.Fatal(err)
	}
	n, err := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode).New(r)
	if err != nil {
		t.Fatal(err)
	}
	node := n.(*p2p.QriNode)
	return node
}

func newTestFS(cafsys cafs.Filestore) qfs.Filesystem {
	return muxfs.NewMux(map[string]qfs.PathResolver{
		"local": localfs.NewFS(),
		"http":  httpfs.NewFS(),
		"cafs":  cafsys,
	})
}

func newTestQriNodeRegClient(t *testing.T, c *regclient.Client) *p2p.QriNode {
	ms := cafs.NewMapstore()
	r, err := repo.NewMemRepo(testPeerProfile, ms, newTestFS(ms), profile.NewMemStore(), c)
	if err != nil {
		t.Fatal(err)
	}
	n, err := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode).New(r)
	if err != nil {
		t.Fatal(err)
	}
	node := n.(*p2p.QriNode)
	return node
}

func newTestDisconnectedQriNode() (*p2p.QriNode, error) {
	ms := cafs.NewMapstore()
	r, err := repo.NewMemRepo(&profile.Profile{}, ms, newTestFS(ms), profile.NewMemStore(), nil)
	if err != nil {
		return nil, err
	}
	p2pconf := config.DefaultP2P()
	// This Node has P2P disabled.
	p2pconf.Enabled = false
	n, err := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode).NewWithConf(r, p2pconf)
	if err != nil {
		return nil, err
	}
	node := n.(*p2p.QriNode)
	return node, err
}

// pulled from actions and base packages
// TODO - we should probably get a test package going at github.com/qri-io/qri/test
func addCitiesDataset(t *testing.T, node *p2p.QriNode) repo.DatasetRef {
	tc, err := dstest.NewTestCaseFromDir(testdataPath("cities"))
	if err != nil {
		t.Fatal(err.Error())
	}
	ds := tc.Input
	ds.Name = tc.Name
	ds.BodyBytes = tc.Body

	ref, err := actions.SaveDataset(node, ds, nil, nil, false, true, false, false, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	return ref
}

func addNowTransformDataset(t *testing.T, node *p2p.QriNode) repo.DatasetRef {
	tc, err := dstest.NewTestCaseFromDir("testdata/now_tf")
	if err != nil {
		t.Fatal(err.Error())
	}
	ds := tc.Input
	ds.Name = tc.Name
	ds.Transform.ScriptPath = "testdata/now_tf/transform.star"

	ref, err := actions.SaveDataset(node, ds, nil, nil, false, true, false, false, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	return ref
}
