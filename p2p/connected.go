package p2p

import (
	"context"
	"encoding/json"
	"time"

	peer "github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

// MtConnected announces a peer connecting to the network
const MtConnected = MsgType("connected")

type pinfoPod struct {
	ID    string
	Addrs []string
}

func (pod pinfoPod) Decode() (
	peer.AddrInfo, error) {
	pi := peer.AddrInfo{}
	id, err := peer.IDB58Decode(pod.ID)
	if err != nil {
		return pi, err
	}
	pi.ID = id

	for _, mstr := range pod.Addrs {
		maddr, err := ma.NewMultiaddr(mstr)
		if err != nil {
			return pi, err
		}
		pi.Addrs = append(pi.Addrs, maddr)
	}

	return pi, nil
}

// AnnounceConnected kicks off a notice to other peers that a profile has connected
func (n *QriNode) AnnounceConnected(ctx context.Context) error {
	pids := n.ConnectedQriPeerIDs()
	log.Debugf("%s AnnounceConnected to %d peers", n.ID, len(pids))

	addrs := []string{}
	for _, ma := range n.host.Addrs() {
		addrs = append(addrs, ma.String())
	}
	ppod := &pinfoPod{
		ID:    n.ID.Pretty(),
		Addrs: addrs,
	}

	data, err := json.Marshal(ppod)
	if err != nil {
		return err
	}

	msg := NewMessage(n.ID, MtConnected, data)

	go func() {
		if err := n.SendMessage(ctx, msg, nil, pids...); err != nil {
			log.Debugf("send profile message error: %s", err.Error())
		}
	}()

	return nil
}

func (n *QriNode) handleConnected(ws *WrappedStream, msg Message) (hangup bool) {
	// hangup = true

	// bail early if we've seen this message before
	if _, ok := n.msgState.Load(msg.ID); ok {
		// log.Debugf("%s already handled msg: %s from %s", n.ID, msg.ID, pid)
		return
	}

	pip := pinfoPod{}
	if err := json.Unmarshal(msg.Body, &pip); err != nil {
		log.Debug(err.Error())
		return
	}
	pinfo, err := pip.Decode()
	if err != nil {
		log.Debug(err.Error())
		return
	}
	n.host.Peerstore().AddAddrs(pinfo.ID, pinfo.Addrs, peerstore.ConnectedAddrTTL)

	// request this peer's profile to connect two node's knowledge of each other
	if _, err := n.RequestProfile(context.TODO(), pinfo.ID); err != nil {
		log.Debug(err.Error())
		return
	}

	// store that we've seen this message, cleaning up after a while
	n.msgState.Store(msg.ID, true)
	go func(id string) {
		<-time.After(time.Minute)
		n.msgState.Delete(id)
	}(msg.ID)

	return
}
