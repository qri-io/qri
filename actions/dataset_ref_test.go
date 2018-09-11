package actions

import (
	"context"
	"testing"

	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
)

func TestResolveDatasetRef(t *testing.T) {
	ctx := context.Background()
	factory := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, factory, 3)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	if err := p2ptest.ConnectNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	// Convert from test nodes to non-test nodes.
	peers := make([]*p2p.QriNode, len(testPeers))
	for i, node := range testPeers {
		peers[i] = node.(*p2p.QriNode)
	}

	// give peer 1 a ref that others don't have
	p, err := peers[1].Repo.Profile()
	if err != nil {
		t.Fatal(err)
	}
	ref := repo.DatasetRef{Peername: p.Peername, Name: "bar", ProfileID: p.ID, Path: "/ipfs/QmXSGsgt8Bn8jepw7beXibYUfWSJVU2SzP3TpkioQVUrmM"}
	if err = peers[1].Repo.PutRef(ref); err != nil {
		t.Fatalf("error putting ref in repo: %s", err.Error())
	}
	expect := "test-repo-1/bar@QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD/ipfs/QmXSGsgt8Bn8jepw7beXibYUfWSJVU2SzP3TpkioQVUrmM"
	in := &repo.DatasetRef{Peername: "test-repo-1", Name: "bar"}

	// TODO - fix this lie
	peers[2].Online = false
	if err := ResolveDatasetRef(peers[2], in); err == nil {
		t.Error("expected offline node to not be able to resolve non-local ref")
	}

	if err := ResolveDatasetRef(peers[0], in); err != nil {
		t.Error(err.Error())
	}
	if in.String() != expect {
		t.Errorf("returned ref mismatch. expected: %s, got: %s", expect, in.String())
	}
}
