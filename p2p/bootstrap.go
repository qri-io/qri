package p2p

import (
	"context"
	"math"
	"math/rand"

	"github.com/ipfs/go-ipfs/core/bootstrap"
	peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// Bootstrap samples a subset of peers & requests their peers list
// This is a naive version of IPFS bootstrapping, which we'll add in once
// qri's settled on a shared-state implementation
func (n *QriNode) Bootstrap(boostrapAddrs []string) {
	peers, err := ParseMultiaddrs(boostrapAddrs)
	if err != nil {
		log.Info("error parsing bootstrap addresses:", err.Error())
		return
	}

	pinfos := toPeerInfos(peers)
	// TODO (ramfox): this randomSubsetOfPeers func is currently always
	// returning the same 4 peers. Right now, I think it's okay to attempt to
	// connect to all 7 of the bootstrap peers
	// when we have more bootstraps in the future, then we can add back
	// only dialing to a random subset
	// for _, p := range randomSubsetOfPeers(pinfos, 4) {
	for _, p := range pinfos {
		go func(p peer.AddrInfo) {
			log.Debugf("boostrapping to: %s", p.ID.Pretty())
			if err := n.host.Connect(context.Background(), p); err != nil {
				log.Infof("error connecting to host: %s", err.Error())
			}
		}(p)
	}
}

// BootstrapIPFS connects this node to standard ipfs nodes for file exchange
func (n *QriNode) BootstrapIPFS() {
	if node, err := n.IPFS(); err == nil {
		if err := node.Bootstrap(bootstrap.DefaultBootstrapConfig); err != nil {
			log.Errorf("IPFS bootsrap error: %s", err.Error())
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
func toPeerInfos(addrs []ma.Multiaddr) []peer.AddrInfo {
	pinfos := make(map[peer.ID]*peer.AddrInfo)
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
			pinfo = new(peer.AddrInfo)
			pinfos[peerid] = pinfo
			pinfo.ID = peerid
		}

		// TODO - support circuit-relay once it normalizes
		split := ma.Split(addr)
		maddr := ma.Join(split[:len(split)-1]...)
		pinfo.Addrs = append(pinfo.Addrs, maddr)
	}

	var peers []peer.AddrInfo
	for _, pinfo := range pinfos {
		peers = append(peers, *pinfo)
	}

	return peers
}

// TODO (ramfox): this is always returning the same bootstrap peers
// since the length of the list of peers that is given are always
// the same
// randomSubsetOfPeers samples up to max from a slice of PeerInfos
func randomSubsetOfPeers(in []peer.AddrInfo, max int) []peer.AddrInfo {
	n := int(math.Min(float64(max), float64(len(in))))
	var out []peer.AddrInfo
	for _, val := range rand.Perm(len(in)) {
		out = append(out, in[val])
		if len(out) >= n {
			break
		}
	}
	return out
}
