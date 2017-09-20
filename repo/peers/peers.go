package peers

import (
	"github.com/libp2p/go-libp2p-peer"
	"github.com/qri-io/qri/repo/profile"
)

// Peers is a store of peer information
// It's named peers to disambiguate from the lib-p2p peerstore
type Peers interface {
	Queryable
	PutPeer(id peer.ID, profile *profile.Profile) error
	DeletePeer(id peer.ID) error
	GetPeer(id peer.ID) (*profile.Profile, error)
}

type Memstore map[peer.ID]*profile.Profile

func (m Memstore) PutPeer(id peer.ID, profile *profile.Profile) error {

}

func (m Memstore) GetPeer(id peer.Id) (*profile.Profile, error) {

}

func (m Memstore) DeletePeer(id peer.ID) error {

}
