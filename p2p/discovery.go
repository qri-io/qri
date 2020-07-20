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
	ctx, cancel := context.WithTimeout(context.Background(), discoveryConnTimeout)
	defer cancel()
	n.Host().Connect(ctx, pinfo)
}
