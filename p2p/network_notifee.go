package p2p

import (
	net "github.com/libp2p/go-libp2p-net"
	ma "github.com/multiformats/go-multiaddr"
)

// a no-op implimentation of the Notifee interface
type networkNotifee struct {
	node *QriNode
}

// Connected is called when a connection opened
func (n networkNotifee) Connected(net net.Network, conn net.Conn) {}

// Disconnectec is called when a connection closed
func (n networkNotifee) Disconnected(net net.Network, conn net.Conn) {}

// OpenedStream is called when a stream opened
func (n networkNotifee) OpenedStream(net net.Network, s net.Stream) {}

// ClosedStream is called when a stream closed
func (n networkNotifee) ClosedStream(net net.Network, s net.Stream) {}

// Listen is called when a network starts listening on an addr
func (n networkNotifee) Listen(net net.Network, addr ma.Multiaddr) {}

// ListenClose is called when a network stops listening on an addr
func (n networkNotifee) ListenClose(net net.Network, addr ma.Multiaddr) {}
