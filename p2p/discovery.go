package p2p

import (
	"context"
	"fmt"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery"
	"time"
)

func (n *QriNode) HandlePeerFound(pinfo pstore.PeerInfo) {
	// fmt.Println("trying peer info: ", pinfo)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := n.Host.Connect(ctx, pinfo); err != nil {
		fmt.Println("Failed to connect to peer found by discovery: ", err)
	}
	n.Host.Peerstore().AddAddr(pinfo.ID, pinfo.Addrs[0], time.Hour)

	p, err := n.repo.Profile()
	if err != nil {
		return
	}

	n.SendMessage(pinfo.Addrs[0], &Message{
		Type:    MtProfile,
		Payload: p,
	})

	fmt.Println("connected to peer: ", pinfo.ID.Pretty())
}

// StartDiscovery initiates peer discovery
func (n *QriNode) StartDiscovery() error {
	service, err := discovery.NewMdnsService(context.Background(), n.Host, time.Second*2)
	if err != nil {
		return err
	}

	service.RegisterNotifee(n)
	n.Discovery = service
	return nil
}
