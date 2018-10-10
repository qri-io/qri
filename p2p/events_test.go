package p2p

import (
	"context"
	"sync"
	"testing"

	"github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
)

func TestRequestEventsList(t *testing.T) {
	ctx := context.Background()
	f := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, f, 5)
	if err != nil {
		t.Errorf("error creating network: %s", err.Error())
		return
	}

	peers := asQriNodes(testPeers)

	for _, p := range peers {
		p.Repo.LogEvent(repo.ETDsCreated, repo.DatasetRef{})
	}

	if err := p2ptest.ConnectNodes(ctx, testPeers); err != nil {
		t.Errorf("error connecting peers: %s", err.Error())
	}

	t.Logf("testing RequestEventsList message with %d peers", len(peers))
	var wg sync.WaitGroup
	for i, p1 := range peers {
		for _, p2 := range peers[i+1:] {
			wg.Add(1)
			go func(p1, p2 *QriNode) {
				defer wg.Done()

				events, err := p1.RequestEventsList(p2.ID, EventsParams{Limit: 10, Offset: 0})
				if err != nil {
					t.Errorf("%s -> %s error: %s", p1.ID.Pretty(), p2.ID.Pretty(), err.Error())
				}
				if events == nil {
					t.Error("profile shouldn't be nil")
					return
				}
				// t.Log(refs)
			}(p1, p2)
		}
	}

	wg.Wait()
}
