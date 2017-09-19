package p2p

import (
	"context"
	"fmt"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery"
	ma "github.com/multiformats/go-multiaddr"
	"time"
)

func (n *QriNode) HandlePeerFound(pinfo pstore.PeerInfo) {
	// fmt.Println("trying peer info: ", pinfo)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	if err := n.Host.Connect(ctx, pinfo); err != nil {
		fmt.Println("Failed to connect to peer found by discovery: ", err)
		return
	}

	// data, _ := pinfo.MarshalJSON()
	// fmt.Println(string(data))

	// p, err := n.repo.Profile()
	// if err != nil {
	// 	return
	// }
	peers, err := n.Repo().Peers()
	if err != nil {
		fmt.Println("error getting peers list: ", err)
		return
	}

	if peers[pinfo.ID.Pretty()] != nil {
		return
	}

	peerAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", pinfo.ID.Pretty()))

	addr := pinfo.Addrs[0].Encapsulate(peerAddr)

	profile, err := n.Repo().Profile()
	if err != nil {
		fmt.Println("error getting node profile info:", err)
		return
	}

	res, err := n.SendMessage(addr.String(), &Message{
		Type:    MtPeerInfo,
		Payload: profile,
	})
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if res.Phase == MpResponse {
		if err := n.handleProfileResponse(pinfo, res); err != nil {
			fmt.Println(err.Error())
		}
	}

	// fmt.Println("connected to peer: ", pinfo.ID.Pretty())
}

// StartDiscovery initiates peer discovery
func (n *QriNode) StartDiscovery() error {
	service, err := discovery.NewMdnsService(context.Background(), n.Host, time.Second*5)
	if err != nil {
		return err
	}

	service.RegisterNotifee(n)
	n.Discovery = service
	return nil
}
