package p2p

import (
	"context"
	"sync"
	"testing"

	"github.com/qri-io/qri/repo"
)

func TestRequestDatasetInfo(t *testing.T) {
	ctx := context.Background()
	peers, err := NewTestDirNetwork(ctx, t)
	if err != nil {
		t.Errorf("error creating network: %s", err.Error())
		return
	}

	if err := connectNodes(ctx, peers); err != nil {
		t.Errorf("error connecting peers: %s", err.Error())
	}

	refs := []repo.DatasetRef{}
	for _, c := range peers {
		if rs, err := c.Repo.References(10, 0); err == nil {
			refs = append(refs, rs...)
		}
	}

	t.Logf("testing RequestDatasetList message with %d peers", len(peers))
	var wg sync.WaitGroup
	for _, p := range peers {
		for _, ref := range refs {
			wg.Add(1)
			go func(p *QriNode, ref repo.DatasetRef) {
				defer wg.Done()
				// ref := repo.DatasetRef{Path: "foo"}
				if err := p.RequestDataset(&ref); err != nil {
					t.Errorf("%s RequestDataset error: %s", p.ID, err.Error())
				}
				if ref.Dataset == nil {
					pro, _ := p.Repo.Profile()
					t.Errorf("%s %s ref.Dataset shouldn't be nil for ds: %s/%s", pro.Peername, p.ID, ref.Peername, ref.Name)
					return
				}
			}(p, ref)
		}
	}

	wg.Wait()
}
