package p2p

import (
	"context"
	"math"
	"math/rand"

	"github.com/ipfs/go-ipfs/core/bootstrap"
	peer "github.com/libp2p/go-libp2p-core/peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

// Bootstrap samples a subset of peers & requests their peers list
// This is a naive version of IPFS bootstrapping, which we'll add in once
// qri's settled on a shared-state implementation
func (n *QriNode) Bootstrap(boostrapAddrs []string, boostrapPeers chan pstore.PeerInfo) {
	peers, err := ParseMultiaddrs(boostrapAddrs)
	if err != nil {
		log.Info("error parsing bootstrap addresses:", err.Error())
		return
	}

	pinfos := toPeerInfos(peers)
	for _, p := range randomSubsetOfPeers(pinfos, 4) {
		go func(p pstore.PeerInfo) {
			log.Debugf("boostrapping to: %s", p.ID.Pretty())
			if err := n.host.Connect(context.Background(), p); err == nil {
				if err = n.UpgradeToQriConnection(p); err != nil && err != ErrQriProtocolNotSupported {
					log.Errorf("error adding peer: %s", err.Error())
				} else {
					boostrapPeers <- p
				}
			} else {
				log.Infof("error connecting to host: %s", err.Error())
			}
		}(p)
	}
}

// BootstrapIPFS connects this node to standard ipfs nodes for file exchange
func (n *QriNode) BootstrapIPFS() {
	if node, err := n.ipfsNode(); err == nil {
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
	n := int(math.Min(float64(max), float64(len(in))))
	var out []pstore.PeerInfo
	for _, val := range rand.Perm(len(in)) {
		out = append(out, in[val])
		if len(out) >= n {
			break
		}
	}
	return out
}
