package floodsub

import (
	"context"

	inet "gx/ipfs/QmRuZnMorqodado1yeTQiv1i9rmtKj29CjPSsBKM7DFXV4/go-libp2p-net"
	ma "gx/ipfs/QmSWLfmj5frN9xVLMMN846dMDriy5wN5jeghUm7aTW3DAG/go-multiaddr"
)

var _ inet.Notifiee = (*PubSubNotif)(nil)

type PubSubNotif PubSub

func (p *PubSubNotif) OpenedStream(n inet.Network, s inet.Stream) {
}

func (p *PubSubNotif) ClosedStream(n inet.Network, s inet.Stream) {
}

func (p *PubSubNotif) Connected(n inet.Network, c inet.Conn) {
	s, err := p.host.NewStream(context.Background(), c.RemotePeer(), ID)
	if err != nil {
		log.Warning("opening new stream to peer: ", err, c.LocalPeer(), c.RemotePeer())
		return
	}

	select {
	case p.newPeers <- s:
	case <-p.ctx.Done():
		s.Close()
	}
}

func (p *PubSubNotif) Disconnected(n inet.Network, c inet.Conn) {
}

func (p *PubSubNotif) Listen(n inet.Network, _ ma.Multiaddr) {
}

func (p *PubSubNotif) ListenClose(n inet.Network, _ ma.Multiaddr) {
}
