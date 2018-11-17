package p2p

import (
	"time"

	peer "gx/ipfs/QmTRhk7cgjUf2gfQ3p2M9KPECNZEW9XUrmHcFCgog4cPgB/go-libp2p-peer"
)

const (
	// MtPing is a ping/pong message
	MtPing = MsgType("ping")
)

// Ping initiates a ping message from peer to a peer.ID
func (n *QriNode) Ping(peerID peer.ID) (time.Duration, error) {
	log.Debugf("Ping %s -> %s", n.ID, peerID)

	replies := make(chan Message)
	defer close(replies)

	now := time.Now()
	ping := NewMessage(n.ID, MtPing, []byte("PING"))
	if err := n.SendMessage(ping, replies, peerID); err != nil {
		return time.Duration(0), err
	}

	<-replies
	return time.Since(now), nil
}

// handlePing handles messages of type MtPing
func (n *QriNode) handlePing(ws *WrappedStream, msg Message) (hangup bool) {
	hangup = true
	switch string(msg.Body) {
	case "PING":
		pong := msg.Update([]byte("PONG"))
		if err := ws.sendMessage(pong); err != nil {
			log.Debug(err.Error())
		}
		return
	case "PONG":
		return
	default:
		log.Debugf("invalid ping messge: %s", string(msg.Body))
		return
	}
}
