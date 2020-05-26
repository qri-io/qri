package regclient

import (
	"context"
	"net/http/httptest"
	"testing"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qri/config"
	testPeers "github.com/qri-io/qri/config/test"
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

	if _, err := (*Client)(nil).ResolveRef(ctx, nil); err != dsref.ErrRefNotFound {
		t.Errorf("expected client to be nil-callable")
	}

	t.Skip("TODO (b5) - registry is not yet spec-compliant")
	dsrefspec.AssertResolverSpec(t, tr.Client, func(ref dsref.Ref, author identity.Author, log *oplog.Log) error {
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
	pk1 := testPeers.GetTestPeerInfo(10).PrivKey

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
