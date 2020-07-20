package p2p

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

// MtPing is a ping/pong message
const MtPing = MsgType("ping")

// Ping initiates a ping message from peer to a peer.ID
func (n *QriNode) Ping(ctx context.Context, peerID peer.ID) (time.Duration, error) {
	log.Debugf("Ping %s -> %s", n.ID, peerID)

	replies := make(chan Message)
	defer close(replies)

	now := time.Now()
	ping := NewMessage(n.ID, MtPing, []byte("PING"))
	if err := n.SendMessage(ctx, ping, replies, peerID); err != nil {
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
