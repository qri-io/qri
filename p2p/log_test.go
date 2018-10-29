package p2p

import (
	"context"
	"sync"
	"testing"

	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p/test"
)

func TestRequestDatasetLog(t *testing.T) {
	ctx := context.Background()
	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestDirNetwork(ctx, factory)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	peers := asQriNodes(testPeers)

	tc, err := dstest.NewTestCaseFromDir("testdata/tim/craigslist")
	if err != nil {
		t.Fatal(err)
	}

	// add a dataset to tim
	ref, err := base.CreateDataset(peers[4].Repo, tc.Name, tc.Input, tc.BodyFile(), true)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("testing RequestDatasetLog message with %d peers", len(peers))
	var wg sync.WaitGroup
	for i, p1 := range peers {
		for _, p2 := range peers[i+1:] {
			wg.Add(1)
			go func(p1, p2 *QriNode) {
				defer wg.Done()

				refs, err := p1.RequestDatasetLog(ref, 100, 0)
				if err != nil {
					t.Errorf("%s -> %s error: %s", p1.ID.Pretty(), p2.ID.Pretty(), err.Error())
				}
				if refs == nil {
					t.Error("profile shouldn't be nil")
					return
				}
				t.Log(refs)
			}(p1, p2)
		}
	}

	wg.Wait()
}
