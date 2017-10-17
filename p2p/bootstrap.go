package p2p

import (
	"context"
	"fmt"
	"math/rand"

	iaddr "github.com/ipfs/go-ipfs/thirdparty/ipfsaddr"
	math2 "github.com/ipfs/go-ipfs/thirdparty/math2"
	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// Bootstrap samples a subset of peers & requests their peers list
// This is a naive version of IPFS bootstrapping, which we'll add in once
// qri's settled on a shared-state implementation
func (n *QriNode) Bootstrap(boostrapAddrs []string) {

	peers, err := ParseBootstrapPeers(boostrapAddrs)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	pinfos := toPeerInfos(peers)

	for _, pi := range randomSubsetOfPeers(pinfos, 4) {
		go func() {
			if err := n.Host.Connect(context.Background(), pi); err == nil {
				fmt.Printf("boostrapping to: %s\n", pi.ID.Pretty())
				n.RequestPeersList(pi.ID)
			} else {
				fmt.Println(err.Error())
			}
		}()
	}
}

// DefaultBootstrapAddresses follows the pattern of IPFS boostrapping off known "gateways".
// This boostrapping is specific to finding qri peers, which are IPFS peers that also
// support the qri protocol.
// (we also perform standard IPFS boostrapping when IPFS networking is enabled, and it's almost always enabled).
// These are addresses to public, qri nodes hosted by qri.
// One day it would be super nice to bootstrap from a stored history & only
// use these for first-round bootstrapping.
var DefaultBootstrapAddresses = []string{
	"/ip4/35.192.124.143/tcp/4001/ipfs/QmXhrdvGRBF5ocvgdLdnMDBMbxJWDngZDBjS2PUYc4ahpb",
}

func ParseBootstrapPeers(addrs []string) ([]iaddr.IPFSAddr, error) {
	peers := make([]iaddr.IPFSAddr, len(addrs))
	var err error
	for i, addr := range addrs {
		peers[i], err = ParseBootstrapPeer(addr)
		if err != nil {
			return nil, err
		}
	}
	return peers, nil
}

func ParseBootstrapPeer(addr string) (iaddr.IPFSAddr, error) {
	ia, err := iaddr.ParseString(addr)
	if err != nil {
		return nil, err
	}
	return iaddr.IPFSAddr(ia), err
}

func toPeerInfos(bpeers []iaddr.IPFSAddr) []pstore.PeerInfo {
	pinfos := make(map[peer.ID]*pstore.PeerInfo)
	for _, bootstrap := range bpeers {
		pinfo, ok := pinfos[bootstrap.ID()]
		if !ok {
			pinfo = new(pstore.PeerInfo)
			pinfos[bootstrap.ID()] = pinfo
			pinfo.ID = bootstrap.ID()
		}

		pinfo.Addrs = append(pinfo.Addrs, bootstrap.Transport())
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
