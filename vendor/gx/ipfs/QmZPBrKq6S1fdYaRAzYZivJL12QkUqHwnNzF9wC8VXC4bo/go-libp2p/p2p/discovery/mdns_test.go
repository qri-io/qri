package discovery

import (
	"context"
	"testing"
	"time"

	bhost "gx/ipfs/QmZPBrKq6S1fdYaRAzYZivJL12QkUqHwnNzF9wC8VXC4bo/go-libp2p/p2p/host/basic"

	host "gx/ipfs/QmRNyPNJGNCaZyYonJj7owciWTsMd9gRfEKmZY3o6xwN3h/go-libp2p-host"
	netutil "gx/ipfs/QmSTbByZ1rJVn8KANcoiLDiPH2pgDaz33uT6JW6B9nMBW5/go-libp2p-netutil"

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
