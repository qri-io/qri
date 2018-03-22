package p2p

import (
	"context"
	"sync"
	"testing"
)

func TestRequestDatasetsList(t *testing.T) {
	ctx := context.Background()
	peers, err := NewTestDirNetwork(ctx, t)
	if err != nil {
		t.Errorf("error creating network: %s", err.Error())
		return
	}

	if err := connectNodes(ctx, peers); err != nil {
		t.Errorf("error connecting peers: %s", err.Error())
	}

	t.Logf("testing RequestDatasetList message with %d peers", len(peers))
	var wg sync.WaitGroup
	for i, p1 := range peers {
		for _, p2 := range peers[i+1:] {
			wg.Add(1)
			go func(p1, p2 *QriNode) {
				defer wg.Done()

				refs, err := p1.RequestDatasetsList(p2.ID, DatasetsListParams{Limit: 10, Offset: 0})
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
