package profile

import (
	"fmt"

	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

var ErrNotFound = fmt.Errorf("profile: not found")

// Store is a store of profile information
type Store interface {
	List() (map[ID]*Profile, error)
	PeerIDs(id ID) ([]peer.ID, error)
	PeernameID(peername string) (ID, error)
	PutProfile(profile *Profile) error
	GetProfile(id ID) (*Profile, error)
	DeleteProfile(id ID) error
}

// MemStore is an in-memory implementation of the profile Store interface
type MemStore map[ID]*Profile

// PutProfile adds a peer to this store
func (m MemStore) PutProfile(profile *Profile) error {
	if profile.ID.String() == "" {
		return fmt.Errorf("profile.ID is required")
	}

	m[profile.ID] = profile
	return nil
}

// PeernameID gives the ID for a given peername
func (m MemStore) PeernameID(peername string) (ID, error) {
	for id, profile := range m {
		if profile.Peername == peername {
			return id, nil
		}
	}
	return "", ErrNotFound
}

// PeerIDs gives the peer.IDs list for a given peername
func (m MemStore) PeerIDs(id ID) ([]peer.ID, error) {
	for proid, profile := range m {
		if id == proid {
			return profile.PeerIDs(), nil
		}
	}

	return nil, ErrNotFound
}

// List hands the full list of peers back
func (m MemStore) List() (map[ID]*Profile, error) {
	res := map[ID]*Profile{}
	for id, p := range m {
		res[id] = p
	}
	return res, nil
}

// GetProfile give's peer info from the store for a given peer.ID
func (m MemStore) GetProfile(id ID) (*Profile, error) {
	if m[id] == nil {
		return nil, ErrNotFound
	}
	return m[id], nil
}

// DeleteProfile removes a peer from this store
func (m MemStore) DeleteProfile(id ID) error {
	delete(m, id)
	return nil
}

// Query grabs a set of peers from this store for given query params
// func (m MemStore) Query(q query.Query) (query.Results, error) {
// 	re := make([]query.Entry, 0, len(m))
// 	for id, v := range m {
// 		re = append(re, query.Entry{Key: id.String(), Value: v})
// 	}
// 	r := query.ResultsWithEntries(q, re)
// 	r = query.NaiveQueryApply(q, r)
// 	return r, nil
// }
