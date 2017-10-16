package p2p

import (
	"fmt"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
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

func (n *QriNode) ConnectedPeers() []string {
	conns := n.Host.Network().Conns()
	peers := make([]string, len(conns))
	for i, c := range conns {
		peers[i] = c.RemotePeer().Pretty()
	}

	return peers
}
