package p2p

import (
	"context"
	"fmt"
	"math/rand"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	ipfscore "gx/ipfs/QmViBzgruNUoLNBnXcx8YWbDNwV8MNGEGKkLo6JGetygdw/go-ipfs/core"
	math2 "gx/ipfs/QmViBzgruNUoLNBnXcx8YWbDNwV8MNGEGKkLo6JGetygdw/go-ipfs/thirdparty/math2"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// DefaultBootstrapAddresses follows the pattern of IPFS boostrapping off known "gateways".
// This boostrapping is specific to finding qri peers, which are IPFS peers that also
// support the qri protocol.
// (we also perform standard IPFS boostrapping when IPFS networking is enabled, and it's almost always enabled).
// These are addresses to public qri nodes hosted by qri, inc.
// One day it would be super nice to bootstrap from a stored history & only
// use these for first-round bootstrapping.
var DefaultBootstrapAddresses = []string{
	"/ip4/130.211.198.23/tcp/4001/ipfs/QmNX9nSos8sRFvqGTwdEme6LQ8R1eJ8EuFgW32F9jjp2Pb", // mojo
}

// Bootstrap samples a subset of peers & requests their peers list
// This is a naive version of IPFS bootstrapping, which we'll add in once
// qri's settled on a shared-state implementation
func (n *QriNode) Bootstrap(boostrapAddrs []string, boostrapPeers chan pstore.PeerInfo) {
	peers, err := ParseMultiaddrs(boostrapAddrs)
	if err != nil {
		n.log.Info("error parsing bootstrap addresses:", err.Error())
		return
	}

	pinfos := toPeerInfos(peers)

	for _, p := range randomSubsetOfPeers(pinfos, 4) {
		go func(p pstore.PeerInfo) {
			n.log.Infof("boostrapping to: %s", p.ID.Pretty())
			if err := n.Host.Connect(context.Background(), p); err == nil {
				if err = n.AddQriPeer(p); err != nil {
					n.log.Infof("error adding peer: %s", err.Error())
				} else {
					boostrapPeers <- p
				}
			} else {
				n.log.Infof("error connecting to host: %s", err.Error())
			}
		}(p)
	}
}

// BootstrapIPFS connects this node to standard ipfs nodes for file exchange
func (n *QriNode) BootstrapIPFS() {
	if node, err := n.IPFSNode(); err == nil {
		if err := node.Bootstrap(ipfscore.DefaultBootstrapConfig); err != nil {
			fmt.Errorf("IPFS bootsrap error: %s", err.Error())
		}
	}
}

// ParseMultiaddrs turns a slice of strings into a slice of Multiaddrs
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

// toPeerInfos turns a slice of multiaddrs into a slice of PeerInfos
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

// randomSubsetOfPeers samples up to max from a slice of PeerInfos
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
