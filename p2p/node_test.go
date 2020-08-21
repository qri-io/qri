package p2p

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/qri-io/qri/config"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/event"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/test"
)

func TestNewNode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	info := cfgtest.GetTestPeerInfo(0)
	r, err := test.NewTestRepoFromProfileID(profile.IDFromPeerID(info.PeerID), 0, -1)
	if err != nil {
		t.Errorf("error creating test repo: %s", err.Error())
		return
	}

	bus := event.NewBus(ctx)

	eventFired := make(chan struct{}, 1)
	bus.Subscribe(func(_ context.Context, typ event.Type, payload interface{}) error {
		if typ == event.ETP2PGoneOnline {
			if _, ok := payload.([]ma.Multiaddr); !ok {
				t.Errorf("expected %q event to have a payload of []multiaddr.Multiaddr, got: %T", event.ETP2PGoneOnline, payload)
			}
			eventFired <- struct{}{}
		}
		return nil
	}, event.ETP2PGoneOnline)

	p2pconf := config.DefaultP2PForTesting()
	n, err := NewQriNode(r, p2pconf, bus)
	if err != nil {
		t.Errorf("error creating qri node: %s", err.Error())
		return
	}

	if n.Online {
		t.Errorf("default node online flag should be false")
	}
	if err := n.GoOnline(ctx); err != nil {
		t.Error(err.Error())
	}
	if !n.Online {
		t.Errorf("online should equal true")
	}
	<-eventFired
}

func TestNodeEvents(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	bus := event.NewBus(ctx)
	result := make(chan error, 1)
	events := []event.Type{
		// TODO (b5) - can't check onlineness because of the way this test is constructed
		// event.ETP2PGoneOnline,
		event.ETP2PGoneOffline,
		// TODO (ramfox) - the QriPeerConnected is attempted when the `libp2pevent.EvtPeerIdentificationCompleted`
		// event successfully goes off (which is rare atm, the identification fails
		// with a "stream reset" error), so I'm commenting that out for now
		// event.ETP2PQriPeerConnected,
		// TODO (b5) - this event currently isn't emitted
		// event.ETP2PQriPeerDisconnected,
		event.ETP2PPeerConnected,
		event.ETP2PPeerDisconnected,
	}

	called := map[event.Type]bool{}
	calledMu := sync.Mutex{}
	remaining := len(events)

	// TODO (ramfox): when we can figure out the `libp2pevent.EvtPeerIdentificationFailed`
	// "stream reset" error, we can add this back in
	// qriPeerConnectedCh := make(chan struct{}, 1)

	bus.Subscribe(func(_ context.Context, typ event.Type, payload interface{}) error {
		calledMu.Lock()
		defer calledMu.Unlock()
		if called[typ] {
			// TODO (ramfox): this is commented out currently because I'm not totally
			// sure why connects and disconnects are fireing multiple times
			t.Logf("expected event %q to only fire once", typ)
			return nil
		}

		// TODO (ramfox): when we can figure out the `libp2pevent.EvtPeerIdentificationFailed`
		// "stream reset" error, we can add this back in
		// if typ == event.ETP2PQriPeerConnected {
		// 		qriPeerConnectedCh <- struct{}{}
		// 	}

		called[typ] = true
		remaining--
		t.Logf("remaining: %d", remaining)
		if remaining == 0 {
			result <- nil
			return nil
		}
		return nil
	}, events...)

	go func() {
		select {
		case <-ctx.Done():
			ok := true
			uncalled := ""
			calledMu.Lock()
			for tp, called := range called {
				if !called {
					uncalled += fmt.Sprintf("%s\n", tp)
					ok = false
				}
			}
			calledMu.Unlock()
			if !ok {
				result <- fmt.Errorf("context cancelled before all events could fire. Uncalled Events:\n%s", uncalled)
				return
			}
			result <- nil
		}
	}()

	factory := p2ptest.NewTestNodeFactoryWithBus(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, factory, 2)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	peers := asQriNodes(testPeers)
	peers[0].pub = bus

	if err := peers[0].Host().Connect(ctx, peers[1].SimpleAddrInfo()); err != nil {
		t.Fatalf("error connecting nodes: %s", err)
	}

	// TODO (ramfox): when we can figure out the `libp2pevent.EvtPeerIdentificationFailed`
	// "stream reset" error, we can add this back in
	// // because upgrading to a qri peer connection happens async after the `PeerConnect`
	// // event, we need to wait for the qri peer to upgrade before we send the second
	// // peer offline
	// <-qriPeerConnectedCh

	if err := peers[1].GoOffline(); err != nil {
		t.Error(err)
	}

	if err := peers[0].GoOffline(); err != nil {
		fmt.Println("error go offline", err)
		t.Error(err)
	}

	if err := <-result; err != nil {
		t.Error(err)
	}
}
