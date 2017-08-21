package p2p

import (
	"fmt"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

func (n *QriNode) PeerIdForMultiaddr(multiaddr string) (peerid peer.ID, err error) {
	addr, err := ma.NewMultiaddr(multiaddr)
	if err != nil {
		err = fmt.Errorf("invalid multiaddr: %s", err.Error())
		return
	}

	pid, err := addr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		return
	}

	peerid, err = peer.IDB58Decode(pid)
	if err != nil {
		return
	}

	// TODO - check peerstore for id
	// if n.Host.Peerstore().Get(peerid, key)

	// Decapsulate the /ipfs/<peerID> part from the target
	// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
	targetPeerAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
	if err != nil {
		return
	}

	targetAddr := addr.Decapsulate(targetPeerAddr)

	// We have a peer ID and a targetAddr so we add it to the peerstore
	n.Host.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)

	return
}
