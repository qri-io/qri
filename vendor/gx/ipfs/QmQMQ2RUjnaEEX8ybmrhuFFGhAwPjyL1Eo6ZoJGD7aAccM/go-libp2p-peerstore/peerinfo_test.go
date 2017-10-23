package peerstore

import (
	"testing"

	ma "gx/ipfs/QmSWLfmj5frN9xVLMMN846dMDriy5wN5jeghUm7aTW3DAG/go-multiaddr"
	peer "gx/ipfs/QmZcUPvPhD1Xvk6mwijYF8AfR3mG31S1YsEfHG4khrFPRr/go-libp2p-peer"
)

func mustAddr(t *testing.T, s string) ma.Multiaddr {
	addr, err := ma.NewMultiaddr(s)
	if err != nil {
		t.Fatal(err)
	}

	return addr
}

func TestPeerInfoMarshal(t *testing.T) {
	a := mustAddr(t, "/ip4/1.2.3.4/tcp/4536")
	b := mustAddr(t, "/ip4/1.2.3.8/udp/7777")
	id, err := peer.IDB58Decode("QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ")
	if err != nil {
		t.Fatal(err)
	}

	pi := &PeerInfo{
		ID:    id,
		Addrs: []ma.Multiaddr{a, b},
	}

	data, err := pi.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	pi2 := new(PeerInfo)
	if err := pi2.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}

	if pi2.ID != pi.ID {
		t.Fatal("ids didnt match after marshal")
	}

	if !pi.Addrs[0].Equal(pi2.Addrs[0]) {
		t.Fatal("wrong addrs")
	}

	if !pi.Addrs[1].Equal(pi2.Addrs[1]) {
		t.Fatal("wrong addrs")
	}

	lgbl := pi2.Loggable()
	if lgbl["peerID"] != id.Pretty() {
		t.Fatal("loggables gave wrong peerID output")
	}
}
