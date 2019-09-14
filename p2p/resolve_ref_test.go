package p2p

import (
	"context"
	"sync"
	"testing"

	"github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
)

func TestResolveDatasetRef(t *testing.T) {
	ctx := context.Background()
	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestDirNetwork(ctx, factory)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	if err = p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	// Convert from test nodes to non-test nodes.
	peers := make([]*QriNode, len(testPeers))
	for i, node := range testPeers {
		peers[i] = node.(*QriNode)
	}

	// give peer 4 a ref that others don't have
	p, err := peers[4].Repo.Profile()
	if err != nil {
		t.Fatal(err)
	}
	ref := repo.DatasetRef{Peername: p.Peername, Name: "bar", ProfileID: p.ID, Path: "/ipfs/QmXSGsgt8Bn8jepw7beXibYUfWSJVU2SzP3TpkioQVUrmM"}
	if err := peers[4].Repo.PutRef(ref); err != nil {
		t.Fatalf("error putting ref in repo: %s", err.Error())
	}
	if err := repo.CanonicalizeDatasetRef(peers[4].Repo, &repo.DatasetRef{Peername: p.Peername, Name: "bar"}); err != nil {
		t.Fatalf("peer must be able to resolve local ref. error: %s", err.Error())
	}

	expect := "tim/bar@QmTPqxoLireaT3xHuqy3shHvqoomeuFfiPn9ySvjt8mbSi/ipfs/QmXSGsgt8Bn8jepw7beXibYUfWSJVU2SzP3TpkioQVUrmM"

	t.Logf("testing ResolveDatasetRef message with %d peers", len(peers))
	var wg sync.WaitGroup
	for i, p := range peers {
		if i != 4 {
			wg.Add(1)
			go func(p *QriNode) {
				defer wg.Done()
				ref := repo.DatasetRef{Peername: "tim", Name: "bar"}
				if err := p.ResolveDatasetRef(ctx, &ref); err != nil {
					t.Errorf("%s ResolveDatasetRef error: %s", p.ID, err.Error())
				}
				if ref.String() != expect {
					pro, _ := p.Repo.Profile()
					t.Errorf("%s %s name mismatch: %s != %s", pro.Peername, p.ID, ref.String(), expect)
					return
				}
			}(p)
		}
	}

	wg.Wait()
}
