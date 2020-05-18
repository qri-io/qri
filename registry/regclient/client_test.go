package regclient

import (
	"context"
	"encoding/base64"
	"net/http/httptest"
	"testing"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/identity"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver/handlers"
	"github.com/qri-io/qri/remote"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestResolveRef(t *testing.T) {
	tr, cleanup := NewTestRunner(t)
	defer cleanup()

	ctx := context.Background()

	if _, err := (*Client)(nil).ResolveRef(ctx, nil); err != dsref.ErrNotFound {
		t.Errorf("expected client to be nil-callable")
	}

	t.Skip("TODO (b5) - registry is not yet spec-compliant")
	dsrefspec.ResolverSpec(t, tr.Client, func(ref dsref.Ref, author identity.Author, log *oplog.Log) error {
		return tr.Reg.Remote.Node().Repo.Logbook().MergeLog(ctx, author, log)
	})
}

type TestRunner struct {
	Reg           registry.Registry
	RegServer     *httptest.Server
	Client        *Client
	ClientPrivKey crypto.PrivKey
}

func NewTestRunner(t *testing.T) (*TestRunner, func()) {
	// build registry
	tmpRepo, err := repotest.NewTempRepo("registry", "regclient-tests", repotest.NewTestCrypto())
	if err != nil {
		t.Fatal(err)
	}

	r, err := tmpRepo.Repo()
	if err != nil {
		t.Fatal(err)
	}

	node, err := p2p.NewQriNode(r, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err)
	}

	rem, err := remote.NewRemote(node, &config.Remote{
		AcceptSizeMax: -1,
		Enabled:       true,
		AllowRemoves:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	reg := registry.Registry{
		Profiles: registry.NewMemProfiles(),
		Search:   &registry.MockSearch{},
		Remote:   rem,
	}
	ts := httptest.NewServer(handlers.NewRoutes(reg))

	// build client
	// base64-encoded Test Private Key, decoded in init
	// peerId: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
	pk1Bytes := []byte(`CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`)
	data, err := base64.StdEncoding.DecodeString(string(pk1Bytes))
	if err != nil {
		t.Fatal(err)
	}
	pk1Bytes = data

	pk1, err := crypto.UnmarshalPrivateKey(pk1Bytes)
	if err != nil {
		t.Fatalf("error unmarshaling private key: %s", err.Error())
	}

	c := NewClient(&Config{
		Location: ts.URL,
	})

	tr := &TestRunner{
		Reg:           reg,
		RegServer:     ts,
		Client:        c,
		ClientPrivKey: pk1,
	}

	cleanup := func() {
		ts.Close()
		tmpRepo.Delete()
	}
	return tr, cleanup
}
