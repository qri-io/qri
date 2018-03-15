package p2p

import (
// "encoding/json"
)

// MtNodes is a request for distributed web nodes associated with this peer
// const MtNodes = MsgType("nodes")

// func (n *QriNode) ipfsNodesHandler(ws *WrappedStream, msg Message) (hangup bool) {
// var addrs []string
// if ipfs, err := n.IPFSNode(); err == nil {
// 	maddrs := ipfs.PeerHost.Addrs()
// 	addrs = make([]string, len(maddrs))
// 	for i, maddr := range maddrs {
// 		addrs[i] = maddr.String()
// 	}
// }

// data, err := json.Marshal(msg.Payload)
// if err != nil {
// 	log.Debug(err.Error())
// 	return true
// }

// return &Message{
//  Type:    MtNodes,
//  Phase:   MpResponse,
//  Payload: addrs,
// }
// 	return
// }

// func (n *QriNode) handleNodesResponse(r *Message) error {
// 	res := []string{}
// 	if err := json.Unmarshal(data, &res); err != nil {
// 		return err
// 	}

// 	for _, addr := range res {
// 		fmt.Println(addr)
// 		a, err := ma.NewMultiaddr(addr)
// 		if err != nil {
// 			return err
// 		}
// 		ipfsv, err := a.ValueForProtocol(ma.P_IPFS)
// 		if err != nil {
// 			return err
// 		}
// 		fmt.Println(ipfsv)
// 	}

// 	return nil
// }
