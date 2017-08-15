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

	data, _ := pinfo.MarshalJSON()
	fmt.Println(string(data))

	n.Host.Peerstore().AddAddr(pinfo.ID, pinfo.Addrs[0], time.Hour)

	// p, err := n.repo.Profile()
	// if err != nil {
	// 	return
	// }

	res, err := n.SendMessage(pinfo.Addrs[0].String(), &Message{
		Type:    MtProfile,
		Payload: nil,
	})
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if res.Phase == MpResponse {
		fmt.Println(res)

		// peers, err := n.repo.Peers()
		// if err != nil {
		// 	return
		// }
	}

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
