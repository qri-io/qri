package p2p

import (
	"context"
	"time"

	peer "github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery"
)

const (
	discoveryConnTimeout = time.Second * 30
	discoveryInterval    = time.Second * 5
)

// setupDiscovery initiates local peer discovery, allocating a discovery service
// if one doesn't exist, then registering to be notified on peer discovery
func (n *QriNode) setupDiscovery(ctx context.Context) error {
	var err error
	if n.Discovery, err = discovery.NewMdnsService(ctx, n.host, discoveryInterval, discovery.ServiceTag); err != nil {
		return err
	}
	// Registering will call n.HandlePeerFound when peers are discovered
	n.Discovery.RegisterNotifee(n)
	return nil
}

// HandlePeerFound deals with the discovery of a peer that may or may not
// support the qri protocol
func (n *QriNode) HandlePeerFound(pinfo peer.AddrInfo) {
	log.Debugf("found peer %s", pinfo.ID)
	time.Sleep(time.Millisecond * 250)
	if len(n.Host().Network().ConnsToPeer(pinfo.ID)) == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), discoveryConnTimeout)
		defer cancel()
		// attempt to connect to anyone we find.
		// TODO (b5) - This might be redundant, but might not be if we're using our
		// own p2p host (as opposed to an IPFS node Host)
		n.Host().Connect(ctx, pinfo)
	}
}
