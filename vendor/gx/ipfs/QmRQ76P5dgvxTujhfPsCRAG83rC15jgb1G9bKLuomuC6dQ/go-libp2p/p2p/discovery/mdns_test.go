package discovery

import (
	"context"
	"testing"
	"time"

	bhost "gx/ipfs/QmRQ76P5dgvxTujhfPsCRAG83rC15jgb1G9bKLuomuC6dQ/go-libp2p/p2p/host/basic"

	host "gx/ipfs/QmaSxYRuMq4pkpBBG2CYaRrPx2z7NmMVEs34b9g61biQA6/go-libp2p-host"
	netutil "gx/ipfs/QmdzuGp4a9pahgXuBeReHdYGUzdVX3FUCwfmWVo5mQfkTi/go-libp2p-netutil"

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
