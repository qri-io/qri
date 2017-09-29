package p2p

import (
	"context"
	"fmt"
	"time"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	discovery "gx/ipfs/QmRQ76P5dgvxTujhfPsCRAG83rC15jgb1G9bKLuomuC6dQ/go-libp2p/p2p/discovery"
)

func (n *QriNode) HandlePeerFound(pinfo pstore.PeerInfo) {
	// fmt.Println("trying peer info: ", pinfo)
	// fmt.Println(pinfo.Addrs)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	if err := n.Host.Connect(ctx, pinfo); err != nil {
		fmt.Println("Failed to connect to peer found by discovery: ", err)
		return
	}

	if profile, _ := n.Repo.Peers().GetPeer(pinfo.ID); profile != nil {
		return
	}
	// peers, err := n.Repo().Peers()
	// if err != nil {
	// 	fmt.Println("error getting peers list: ", err)
	// 	return
	// }

	// if peers[pinfo.ID.Pretty()] != nil {
	// 	return
	// }

	profile, err := n.Repo.Profile()
	if err != nil {
		fmt.Println("error getting node profile info:", err)
		return
	}

	res, err := n.SendMessage(pinfo, &Message{
		Type:    MtPeerInfo,
		Payload: profile,
	})
	if err != nil {
		fmt.Println("send profile message error:", err.Error())
		return
	}

	if res.Phase == MpResponse {
		if err := n.handleProfileResponse(pinfo, res); err != nil {
			fmt.Println("profile response error", err.Error())
		}
	}

	res, err = n.SendMessage(pinfo, &Message{
		Type: MtDatasets,
		Payload: &DatasetsReqParams{
			Limit:  30,
			Offset: 0,
		},
	})
	if err != nil {
		fmt.Println("send message error", err.Error())
		return
	}
	if res.Phase == MpResponse {
		if err := n.handleDatasetsResponse(pinfo, res); err != nil {
			fmt.Println("dataset response error:", err.Error())
		}
	}

	// fmt.Println("connected to peer: ", pinfo.ID.Pretty())
}

// StartDiscovery initiates peer discovery
func (n *QriNode) StartDiscovery() error {
	if n.Discovery == nil {
		service, err := discovery.NewMdnsService(context.Background(), n.Host, time.Second*5)
		if err != nil {
			return err
		}
		n.Discovery = service
	}
	n.Discovery.RegisterNotifee(n)
	return nil
}
