package routedhost

import (
	"context"
	"fmt"
	"time"

	host "gx/ipfs/QmbzbRyd22gcW92U1rA2yKagB3myMYhk45XBknJ49F9XWJ/go-libp2p-host"

	pstore "gx/ipfs/QmQMQ2RUjnaEEX8ybmrhuFFGhAwPjyL1Eo6ZoJGD7aAccM/go-libp2p-peerstore"
	inet "gx/ipfs/QmRuZnMorqodado1yeTQiv1i9rmtKj29CjPSsBKM7DFXV4/go-libp2p-net"
	ma "gx/ipfs/QmSWLfmj5frN9xVLMMN846dMDriy5wN5jeghUm7aTW3DAG/go-multiaddr"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	lgbl "gx/ipfs/QmTcfnDHimxBJqx6utpnWqVHdvyquXgkwAvYt4zMaJMKS2/go-libp2p-loggables"
	msmux "gx/ipfs/QmTnsezaB1wWNRHeHnYrm8K4d5i9wtyj3GsqjC3Rt5b5v5/go-multistream"
	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	peer "gx/ipfs/QmZcUPvPhD1Xvk6mwijYF8AfR3mG31S1YsEfHG4khrFPRr/go-libp2p-peer"
)

var log = logging.Logger("routedhost")

// AddressTTL is the expiry time for our addresses.
// We expire them quickly.
const AddressTTL = time.Second * 10

// RoutedHost is a p2p Host that includes a routing system.
// This allows the Host to find the addresses for peers when
// it does not have them.
type RoutedHost struct {
	host  host.Host // embedded other host.
	route Routing
}

type Routing interface {
	FindPeer(context.Context, peer.ID) (pstore.PeerInfo, error)
}

func Wrap(h host.Host, r Routing) *RoutedHost {
	return &RoutedHost{h, r}
}

// Connect ensures there is a connection between this host and the peer with
// given peer.ID. See (host.Host).Connect for more information.
//
// RoutedHost's Connect differs in that if the host has no addresses for a
// given peer, it will use its routing system to try to find some.
func (rh *RoutedHost) Connect(ctx context.Context, pi pstore.PeerInfo) error {
	// first, check if we're already connected.
	if len(rh.Network().ConnsToPeer(pi.ID)) > 0 {
		return nil
	}

	// if we were given some addresses, keep + use them.
	if len(pi.Addrs) > 0 {
		rh.Peerstore().AddAddrs(pi.ID, pi.Addrs, pstore.TempAddrTTL)
	}

	// Check if we have some addresses in our recent memory.
	addrs := rh.Peerstore().Addrs(pi.ID)
	if len(addrs) < 1 {

		// no addrs? find some with the routing system.
		pi2, err := rh.route.FindPeer(ctx, pi.ID)
		if err != nil {
			return err // couldnt find any :(
		}
		if pi2.ID != pi.ID {
			err = fmt.Errorf("routing failure: provided addrs for different peer")
			logRoutingErrDifferentPeers(ctx, pi.ID, pi2.ID, err)
			return err
		}
		addrs = pi2.Addrs
	}

	// if we're here, we got some addrs. let's use our wrapped host to connect.
	pi.Addrs = addrs
	return rh.host.Connect(ctx, pi)
}

func logRoutingErrDifferentPeers(ctx context.Context, wanted, got peer.ID, err error) {
	lm := make(lgbl.DeferredMap)
	lm["error"] = err
	lm["wantedPeer"] = func() interface{} { return wanted.Pretty() }
	lm["gotPeer"] = func() interface{} { return got.Pretty() }
	log.Event(ctx, "routingError", lm)
}

func (rh *RoutedHost) ID() peer.ID {
	return rh.host.ID()
}

func (rh *RoutedHost) Peerstore() pstore.Peerstore {
	return rh.host.Peerstore()
}

func (rh *RoutedHost) Addrs() []ma.Multiaddr {
	return rh.host.Addrs()
}

func (rh *RoutedHost) Network() inet.Network {
	return rh.host.Network()
}

func (rh *RoutedHost) Mux() *msmux.MultistreamMuxer {
	return rh.host.Mux()
}

func (rh *RoutedHost) SetStreamHandler(pid protocol.ID, handler inet.StreamHandler) {
	rh.host.SetStreamHandler(pid, handler)
}

func (rh *RoutedHost) SetStreamHandlerMatch(pid protocol.ID, m func(string) bool, handler inet.StreamHandler) {
	rh.host.SetStreamHandlerMatch(pid, m, handler)
}

func (rh *RoutedHost) RemoveStreamHandler(pid protocol.ID) {
	rh.host.RemoveStreamHandler(pid)
}

func (rh *RoutedHost) NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (inet.Stream, error) {
	return rh.host.NewStream(ctx, p, pids...)
}
func (rh *RoutedHost) Close() error {
	// no need to close IpfsRouting. we dont own it.
	return rh.host.Close()
}

var _ (host.Host) = (*RoutedHost)(nil)
