package p2p

import (
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// ClosestConnectedPeers checks if a peer is connected, and if so adds it to the top
// of a slice cap(max) of peers to try to connect to
// TODO - In the future we'll use a few tricks to improve on just iterating the list
// at a bare minimum we should grab a randomized set of peers
func (n *QriNode) ClosestConnectedPeers(id peer.ID, max int) (pid []peer.ID) {
	added := 0

	if len(n.Host.Network().ConnsToPeer(id)) > 0 {
		added++
		pid = append(pid, id)
	}

	for _, conn := range n.Host.Network().Conns() {
		pid = append(pid, conn.RemotePeer())
		added++
		if added == max {
			break
		}
	}

	return
}
