package p2p

import (
	"context"
	"fmt"
	"github.com/qri-io/qri/repo/profile"
	"sync"
	"testing"

	"github.com/qri-io/qri/repo"
)

func TestAnnounceDatasetChanges(t *testing.T) {
	ctx := context.Background()
	peers, err := NewTestNetwork(ctx, t, 5)
	if err != nil {
		t.Errorf("error creating network: %s", err.Error())
		return
	}

	if err := connectNodes(ctx, peers); err != nil {
		t.Errorf("error connecting peers: %s", err.Error())
	}

	t.Logf("testing AnnounceDatasetChanges message with %d peers", len(peers))
	var wg sync.WaitGroup
	for i, p := range peers {
		wg.Add(1)

		r := make(chan Message)
		p.ReceiveMessages(r)

		go func(p *QriNode) {
			for {
				msg := <-r

				if msg.Type != MtDatasetChanges {
					t.Error("expected only dataset_changes messages")
				}

				count, err := p.Repo.RefCache().RefCount()
				if err != nil {
					t.Errorf("%s, error getting RefCount: %s", p.ID, err.Error())
				}

				if count == len(peers)-1 {
					wg.Done()
					return
				}
			}
		}(p)

		go func(i int, p *QriNode) {
			if err := p.AnnounceDatasetChanges(DatasetChanges{
				Created: []string{
					repo.DatasetRef{Peername: fmt.Sprintf("peer-%d", i), PeerID: p.ID.Pretty(), Name: fmt.Sprintf("dataset-%d", i), Path: fmt.Sprintf("/ipfs/QmeLid2tvSZvuUDStCbP3zvzDk1977JmUcFxD9tYjwwmY%d", i)}.String(),
				},
			}); err != nil {
				t.Errorf("%s error: %s", p.ID.Pretty(), err.Error())
			}
		}(i, p)
	}

	wg.Wait()
}

func TestSelfReplication(t *testing.T) {

	ctx := context.Background()
	peers, err := NewTestDirNetwork(ctx, t)
	if err != nil {
		t.Error(err.Error())
		return
	}

	box1 := peers[0]

	pro, err := box1.Repo.Profile()
	if err != nil {
		t.Error(err.Error())
		return
	}

	pid, err := profile.IDB58Decode(pro.ID)
	if err != nil {
		t.Error(err.Error())
		return
	}

	box2Repo, err := NewTestRepo(pid)
	if err != nil {
		t.Error(err.Error())
		return
	}

	box2, err := newTestQriNode(box2Repo, t)
	if err != nil {
		t.Error(err.Error())
		return
	}

	// pro, _ := box2.Repo.Profile()
	// log.Debugf("box2 ids: %s %s", box2.ID.Pretty(), pro.ID)

	peers = append(peers, box2)
	if err := connectNodes(ctx, peers); err != nil {
		t.Error(err.Error())
		return
	}

	rcpre, err := box2.Repo.RefCount()
	if err != nil {
		t.Error(err.Error())
		return
	}

	t.Logf("testing %d peers", len(peers))
	var wg sync.WaitGroup

	for i, p := range peers {
		wg.Add(1)

		r := make(chan Message)
		p.ReceiveMessages(r)

		go func(p *QriNode) {
			for {
				msg := <-r

				if msg.Type != MtDatasetChanges {
					t.Error("expected only dataset_changes messages")
				}

				count, err := p.Repo.RefCache().RefCount()
				if err != nil {
					t.Errorf("%s, error getting RefCount: %s", p.ID, err.Error())
				}

				// log.Debugf("%s has %d refs", p.ID, count)

				if count == len(peers)-2 {
					// log.Debugf("%s is done", p.ID)
					wg.Done()
					return
				}
			}
		}(p)

		go func(i int, p *QriNode) {
			refs, err := p.Repo.References(1, 0)
			if err != nil {
				t.Errorf("%s error: %s", p.ID.Pretty(), err.Error())
			}

			if len(refs) > 0 {
				if err := p.AnnounceDatasetChanges(DatasetChanges{
					Created: []string{
						refs[0].String(),
					},
				}); err != nil {
					t.Errorf("%s error: %s", p.ID.Pretty(), err.Error())
				}
			}

		}(i, p)
	}

	wg.Wait()

	rcpost, err := box2.Repo.RefCount()
	if err != nil {
		t.Error(err.Error())
		return
	}

	if rcpre+1 != rcpost {
		t.Errorf("expected box2 refcount to increment by 1. expected: %d, got: %d", rcpre+1, rcpost)
	}

}
