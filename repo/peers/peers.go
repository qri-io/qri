package peers

import (
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/qri-io/qri/repo/profile"
)

// Peers is a store of peer information
// It's named peers to disambiguate from the lib-p2p peerstore
type Peers interface {
	Query(query.Query) (query.Results, error)
	PutPeer(id peer.ID, profile *profile.Profile) error
	DeletePeer(id peer.ID) error
	GetPeer(id peer.ID) (*profile.Profile, error)
}

// Memstore is an in-memory implementation of the Peers interface
type Memstore map[peer.ID]*profile.Profile

func (m Memstore) PutPeer(id peer.ID, profile *profile.Profile) error {
	m[id] = profile
	return nil
}

func (m Memstore) GetPeer(id peer.ID) (*profile.Profile, error) {
	if m[id] == nil {
		return nil, datastore.ErrNotFound
	}
	return m[id], nil
}

func (m Memstore) DeletePeer(id peer.ID) error {
	delete(m, id)
	return nil
}

func (m Memstore) Query(q query.Query) (query.Results, error) {
	re := make([]query.Entry, 0, len(m))
	for id, v := range m {
		re = append(re, query.Entry{Key: id.String(), Value: v})
	}
	r := query.ResultsWithEntries(q, re)
	r = query.NaiveQueryApply(q, r)
	return r, nil
}
