package p2p

import (
	"context"
	"testing"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/qri-io/qri/config"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/event"
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
