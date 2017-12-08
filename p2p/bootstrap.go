package p2p

import (
	"context"
	"math/rand"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	math2 "gx/ipfs/QmdKL1GVaUaDVt3JUWiYQSLYRsJMym2KRWxsiXAeEU6pzX/go-ipfs/thirdparty/math2"
)

// DefaultBootstrapAddresses follows the pattern of IPFS boostrapping off known "gateways".
// This boostrapping is specific to finding qri peers, which are IPFS peers that also
// support the qri protocol.
// (we also perform standard IPFS boostrapping when IPFS networking is enabled, and it's almost always enabled).
// These are addresses to public, qri nodes hosted by qri.
// One day it would be super nice to bootstrap from a stored history & only
// use these for first-round bootstrapping.
var DefaultBootstrapAddresses = []string{
	"/ip4/35.192.124.143/tcp/4001/ipfs/QmXNqD5ATi1ejL4HNzUzDyeWn46hHgJTqA26JmYiUWERcb",
}

// Bootstrap samples a subset of peers & requests their peers list
// This is a naive version of IPFS bootstrapping, which we'll add in once
// qri's settled on a shared-state implementation
func (n *QriNode) Bootstrap(boostrapAddrs []string) {
	peers, err := ParseMultiaddrs(boostrapAddrs)
	if err != nil {
		n.log.Info("error parsing bootstrap addresses:", err.Error())
		return
	}

	pinfos := toPeerInfos(peers)

	for _, pi := range randomSubsetOfPeers(pinfos, 4) {
		go func() {
			if err := n.Host.Connect(context.Background(), pi); err == nil {
				n.log.Infof("boostrapping to: %s", pi.ID.Pretty())
				if err = n.AddQriPeer(pi); err != nil {
					n.log.Infof("error adding peer: %s", err.Error())
				}
				n.RequestPeersList(pi.ID)
			} else {
				n.log.Infof("error connecting to host: %s", err.Error())
			}
		}()
	}
}

func ParseMultiaddrs(addrs []string) (maddrs []ma.Multiaddr, err error) {
	maddrs = make([]ma.Multiaddr, len(addrs))
	for i, adr := range addrs {
		maddrs[i], err = ma.NewMultiaddr(adr)
		if err != nil {
			return
		}
	}
	return
}

func toPeerInfos(addrs []ma.Multiaddr) []pstore.PeerInfo {
	pinfos := make(map[peer.ID]*pstore.PeerInfo)
	for _, addr := range addrs {
		pid, err := addr.ValueForProtocol(ma.P_IPFS)
		if err != nil {
			return nil
		}
		peerid, err := peer.IDB58Decode(pid)
		if err != nil {
			return nil
		}

		pinfo, ok := pinfos[peerid]
		if !ok {
			pinfo = new(pstore.PeerInfo)
			pinfos[peerid] = pinfo
			pinfo.ID = peerid
		}

		// TODO - support circuit-relay once it normalizes
		split := ma.Split(addr)
		maddr := ma.Join(split[:len(split)-1]...)
		pinfo.Addrs = append(pinfo.Addrs, maddr)
	}

	var peers []pstore.PeerInfo
	for _, pinfo := range pinfos {
		peers = append(peers, *pinfo)
	}

	return peers
}

func randomSubsetOfPeers(in []pstore.PeerInfo, max int) []pstore.PeerInfo {
	n := math2.IntMin(max, len(in))
	var out []pstore.PeerInfo
	for _, val := range rand.Perm(len(in)) {
		out = append(out, in[val])
		if len(out) >= n {
			break
		}
	}
	return out
}
