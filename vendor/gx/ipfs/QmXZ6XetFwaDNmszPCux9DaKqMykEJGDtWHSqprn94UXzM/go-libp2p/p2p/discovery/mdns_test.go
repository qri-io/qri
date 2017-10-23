package discovery

import (
	"context"
	"testing"
	"time"

	bhost "gx/ipfs/QmXZ6XetFwaDNmszPCux9DaKqMykEJGDtWHSqprn94UXzM/go-libp2p/p2p/host/basic"

	host "gx/ipfs/QmW8Rgju5JrSMHP7RDNdiwwXyenRqAbtSaPfdQKQC7ZdH6/go-libp2p-host"
	netutil "gx/ipfs/QmZG4W8GR9FpC4z69Vab9ENtEoxKjDnTym5oa7Q3Yr7P4o/go-libp2p-netutil"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
)

type DiscoveryNotifee struct {
	h host.Host
}

func (n *DiscoveryNotifee) HandlePeerFound(pi pstore.PeerInfo) {
	n.h.Connect(context.Background(), pi)
}

func TestMdnsDiscovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := bhost.New(netutil.GenSwarmNetwork(t, ctx))
	b := bhost.New(netutil.GenSwarmNetwork(t, ctx))

	sa, err := NewMdnsService(ctx, a, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	sb, err := NewMdnsService(ctx, b, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	_ = sb

	n := &DiscoveryNotifee{a}

	sa.RegisterNotifee(n)

	time.Sleep(time.Second * 2)

	err = a.Connect(ctx, pstore.PeerInfo{ID: b.ID()})
	if err != nil {
		t.Fatal(err)
	}
}
