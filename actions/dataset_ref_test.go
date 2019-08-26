package actions

import (
	"context"
	"testing"

	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
)

func TestResolveDatasetRef(t *testing.T) {
	ctx := context.Background()
	factory := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, factory, 3)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	// Convert from test nodes to qri nodes.
	peers := make([]*p2p.QriNode, len(testPeers))
	for i, node := range testPeers {
		peers[i] = node.(*p2p.QriNode)
	}

	if _, err := ResolveDatasetRef(peers[0], nil, "", &repo.DatasetRef{}); err != repo.ErrEmptyRef {
		t.Errorf("expected repo.ErrEmptRef, got: %s", err)
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
	local, err := ResolveDatasetRef(peers[2], nil, "", in)
	if err == nil {
		t.Error("expected offline node to not be able to resolve non-local ref")
	}
	if local != false {
		t.Error("expected local to equal false")
	}

	if local, err = ResolveDatasetRef(peers[0], nil, "", in); err != nil {
		t.Error(err.Error())
	}
	if local != false {
		t.Error("expected local to equal false")
	}
	if in.String() != expect {
		t.Errorf("returned ref mismatch. expected: %s, got: %s", expect, in.String())
	}

	if local, err = ResolveDatasetRef(peers[1], nil, "", in); err != nil {
		t.Error(err.Error())
	}
	if local != true {
		t.Error("expected local to equal true")
	}
}
