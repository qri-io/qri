package profile

import (
	"fmt"
	"sync"

	"gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

// ErrNotFound is the not found err for the profile package
var ErrNotFound = fmt.Errorf("profile: not found")

// Store is a store of profile information
type Store interface {
	List() (map[ID]*Profile, error)
	PeerIDs(id ID) ([]peer.ID, error)
	PeernameID(peername string) (ID, error)
	PutProfile(profile *Profile) error
	GetProfile(id ID) (*Profile, error)
	PeerProfile(id peer.ID) (*Profile, error)
	DeleteProfile(id ID) error
}

// MemStore is an in-memory implementation of the profile Store interface
type MemStore struct {
	sync.RWMutex
	store map[ID]*Profile
}

// NewMemStore allocates a MemStore
func NewMemStore() Store {
	return &MemStore{
		store: map[ID]*Profile{},
	}
}

// PutProfile adds a peer to this store
func (m MemStore) PutProfile(profile *Profile) error {
	if profile.ID.String() == "" {
		return fmt.Errorf("profile.ID is required")
	}

	m.Lock()
	m.store[profile.ID] = profile
	m.Unlock()
	return nil
}

// PeernameID gives the ID for a given peername
func (m MemStore) PeernameID(peername string) (ID, error) {
	m.RLock()
	defer m.RUnlock()

	for id, profile := range m.store {
		if profile.Peername == peername {
			return id, nil
		}
	}
	return "", ErrNotFound
}

// PeerProfile returns profile data for a given peer.ID
// TODO - this func implies that peer.ID's are only ever connected to the same
// profile. That could cause trouble.
func (m MemStore) PeerProfile(id peer.ID) (*Profile, error) {
	m.RLock()
	defer m.RUnlock()

	for _, profile := range m.store {
		if _, ok := profile.Addresses[id.Pretty()]; ok {
			return profile, nil
		}
	}

	return nil, ErrNotFound
}

// PeerIDs gives the peer.IDs list for a given peername
func (m MemStore) PeerIDs(id ID) ([]peer.ID, error) {
	m.RLock()
	defer m.RUnlock()

	for proid, profile := range m.store {
		if id == proid {
			return profile.PeerIDs(), nil
		}
	}

	return nil, ErrNotFound
}

// List hands the full list of peers back
func (m MemStore) List() (map[ID]*Profile, error) {
	m.RLock()
	defer m.RUnlock()

	res := map[ID]*Profile{}
	for id, p := range m.store {
		res[id] = p
	}
	return res, nil
}

// GetProfile give's peer info from the store for a given peer.ID
func (m MemStore) GetProfile(id ID) (*Profile, error) {
	m.RLock()
	defer m.RUnlock()

	if m.store[id] == nil {
		return nil, ErrNotFound
	}
	return m.store[id], nil
}

// DeleteProfile removes a peer from this store
func (m MemStore) DeleteProfile(id ID) error {
	m.Lock()
	delete(m.store, id)
	m.Unlock()

	return nil
}
