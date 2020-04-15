package p2p

import (
	"context"
	"fmt"
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
	var (
		bus    event.Bus
		result = make(chan error)
		events = []event.Type{
			// TODO (b5) - can't check onlineness because of the way this test is constructed
			// event.ETP2PGoneOnline,
			event.ETP2PGoneOffline,
			event.ETP2PQriPeerConnected,
			// TODO (b5) - this event currently isn't emitted
			// event.ETP2PQriPeerDisconnected,
			event.ETP2PPeerConnected,
			event.ETP2PPeerDisconnected,
		}
	)

	ctx, done := context.WithTimeout(context.Background(), time.Second)
	defer done()

	t.Errorf("1")
	bus = event.NewBus(ctx)
	called := map[event.Type]bool{}
	for _, t := range events {
		called[t] = false
	}
	remaining := len(events)

	bus.Subscribe(func(_ context.Context, typ event.Type, payload interface{}) error {
		if called[typ] {
			t.Errorf("expected event %q to only fire once", typ)
		}

		t.Errorf("event fired: %s", typ)
		called[typ] = true
		remaining--
		if remaining == 0 {
			result <- nil
			return nil
		}
		return nil
	}, events...)

	go func() {
		t.Errorf("2")
		select {
		case <-ctx.Done():
			ok := true
			uncalled := ""
			for tp, called := range called {
				if !called {
					uncalled += fmt.Sprintf("%s\n", tp)
					ok = false
				}
			}
			if !ok {
				result <- fmt.Errorf("context cancelled before all events could fire. Uncalled Events:\n%s", uncalled)
				return
			}
			result <- nil
		}
	}()

	t.Errorf("3")
	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, factory, 2)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	peers := asQriNodes(testPeers)
	peers[0].pub = bus
	t.Errorf("4")

	if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	t.Errorf("5")
	if err := peers[1].GoOffline(); err != nil {
		t.Error(err)
	}

	t.Errorf("6")
	if err := peers[0].GoOffline(); err != nil {
		t.Error(err)
	}

	t.Errorf("6")
	if err := <-result; err != nil {
		t.Error(err)
	}
	t.Errorf("7")
}
