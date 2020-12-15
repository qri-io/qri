package logsync

import (
	"context"
	"testing"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	pstoremem "github.com/libp2p/go-libp2p-peerstore/pstoremem"
	ma "github.com/multiformats/go-multiaddr"
)

func TestP2PLogsync(t *testing.T) {
	t.Skip("TODO(dustmop): validating profileID in logbook data causes this to hang")

	tr, cleanup := newTestRunner(t)
	defer cleanup()

	aHost := p2pHost(tr.Ctx, tr.APrivKey, t)
	bHost := p2pHost(tr.Ctx, tr.BPrivKey, t)

	lsA := New(tr.A, func(o *Options) {
		o.Libp2pHost = aHost
	})

	lsB := New(tr.B, func(o *Options) {
		o.Libp2pHost = bHost
	})

	// connect a & b
	if err := aHost.Connect(tr.Ctx, peer.AddrInfo{ID: bHost.ID(), Addrs: bHost.Addrs()}); err != nil {
		t.Fatal(err)
	}

	// make some logs on A
	worldBankRef, err := writeWorldBankLogs(tr.Ctx, tr.A)
	if err != nil {
		t.Fatal(err)
	}

	// pull logs to B from A
	aID, err := tr.A.ActivePeerID(tr.Ctx)
	if err != nil {
		t.Fatal(err)
	}

	pull, err := lsB.NewPull(worldBankRef, aID)
	if err != nil {
		t.Error(err)
	}
	pull.Merge = true

	if _, err := pull.Do(tr.Ctx); err != nil {
		t.Error(err)
	}

	items, err := tr.B.Items(tr.Ctx, worldBankRef, 0, 10)
	if err != nil {
		t.Errorf("expected no error fetching dslog items after pull. got: %s", err)
	}
	if len(items) == 0 {
		t.Errorf("expected some length of logs. got: %d", len(items))
	}

	// add moar logs to A
	nasdaqRef, err := writeNasdaqLogs(tr.Ctx, tr.A)
	if err != nil {
		t.Fatal(err)
	}

	// push logs from A to B
	bID, err := tr.B.ActivePeerID(tr.Ctx)
	if err != nil {
		t.Fatal(err)
	}

	push, err := lsA.NewPush(nasdaqRef, bID)
	if err != nil {
		t.Fatal(err)
	}

	if err := push.Do(tr.Ctx); err != nil {
		t.Fatal(err)
	}

	items, err = tr.B.Items(tr.Ctx, nasdaqRef, 0, 10)
	if err != nil {
		t.Errorf("expected no error fetching dslog items after pull. got: %s", err)
	}
	if len(items) == 0 {
		t.Errorf("expected some length of logs. got: %d", len(items))
	}

	// A request B removes nasdaq
	if err := lsA.DoRemove(tr.Ctx, nasdaqRef, bID); err != nil {
		t.Errorf("unexpected error doing remove request: %s", err)
	}
	if _, err = tr.B.Items(tr.Ctx, nasdaqRef, 0, 10); err == nil {
		t.Errorf("expected error fetching dslog items. got nil")
	}
}

// makeBasicHost creates a LibP2P host from a NodeCfg
func p2pHost(ctx context.Context, pk crypto.PrivKey, t *testing.T) host.Host {
	pid, err := peer.IDFromPrivateKey(pk)
	if err != nil {
		t.Fatal(err)
	}

	ps := pstoremem.NewPeerstore()
	ps.AddPrivKey(pid, pk)
	ps.AddPubKey(pid, pk.GetPublic())

	addr, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")

	opts := []libp2p.Option{
		libp2p.Identity(pk),
		libp2p.ListenAddrs(addr),
		libp2p.Peerstore(ps),
	}

	host, err := libp2p.New(ctx, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return host
}
