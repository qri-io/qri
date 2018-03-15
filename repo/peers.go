package repo

import (
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/qri/repo/profile"

	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// Peers is a store of peer information
// It's named peers to disambiguate from the lib-p2p peerstore
type Peers interface {
	List() (map[string]*profile.Profile, error)
	Query(query.Query) (query.Results, error)
	IPFSPeerID(peername string) (peer.ID, error)
	GetID(peername string) (peer.ID, error)
	PutPeer(id peer.ID, profile *profile.Profile) error
	GetPeer(id peer.ID) (*profile.Profile, error)
	DeletePeer(id peer.ID) error
}

// QueryPeers wraps a call to Query, transforming responses to a slice of
// Profile pointers
func QueryPeers(ps Peers, q query.Query) ([]*profile.Profile, error) {
	i := 0
	peers := []*profile.Profile{}
	results, err := ps.Query(q)
	if err != nil {
		return nil, err
	}

	if q.Limit != 0 {
		peers = make([]*profile.Profile, q.Limit)
	}

	for res := range results.Next() {
		p, ok := res.Value.(*profile.Profile)
		if !ok {
			return nil, fmt.Errorf("query returned the wrong type, expected a profile pointer")
		}
		if q.Limit != 0 {
			peers[i] = p
		} else {
			peers = append(peers, p)
		}
		i++
	}

	if q.Limit != 0 {
		peers = peers[:i]
	}

	return peers, nil
}

// MemPeers is an in-memory implementation of the Peers interface
type MemPeers map[peer.ID]*profile.Profile

// PutPeer adds a peer to this store
func (m MemPeers) PutPeer(id peer.ID, profile *profile.Profile) error {
	m[id] = profile
	return nil
}

// GetID gives the peer.ID for a given peername
func (m MemPeers) GetID(peername string) (peer.ID, error) {
	for id, profile := range m {
		if profile.Peername == peername {
			return id, nil
		}
	}
	return "", ErrNotFound
}

// IPFSPeerID gives the IPFS peer.ID for a given peername
func (m MemPeers) IPFSPeerID(peername string) (peer.ID, error) {
	for id, profile := range m {
		if profile.Peername == peername {
			if ipfspid, err := profile.IPFSPeerID(); err == nil {
				return ipfspid, nil
			}
			return id, nil
		}
	}

	return "", ErrNotFound
}

// List hands the full list of peers back
func (m MemPeers) List() (map[string]*profile.Profile, error) {
	res := map[string]*profile.Profile{}
	for id, p := range m {
		res[id.Pretty()] = p
	}
	return res, nil
}

// GetPeer give's peer info from the store for a given peer.ID
func (m MemPeers) GetPeer(id peer.ID) (*profile.Profile, error) {
	if m[id] == nil {
		return nil, datastore.ErrNotFound
	}
	return m[id], nil
}

// DeletePeer removes a peer from this store
func (m MemPeers) DeletePeer(id peer.ID) error {
	delete(m, id)
	return nil
}

// Query grabs a set of peers from this store for given query params
func (m MemPeers) Query(q query.Query) (query.Results, error) {
	re := make([]query.Entry, 0, len(m))
	for id, v := range m {
		re = append(re, query.Entry{Key: id.String(), Value: v})
	}
	r := query.ResultsWithEntries(q, re)
	r = query.NaiveQueryApply(q, r)
	return r, nil
}
