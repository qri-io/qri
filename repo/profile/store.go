package profile

import (
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
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
	sync.Mutex
	store map[ID]*Profile
}

// NewMemStore allocates a MemStore
func NewMemStore() Store {
	return &MemStore{
		store: map[ID]*Profile{},
	}
}

// PutProfile adds a peer to this store
func (m *MemStore) PutProfile(profile *Profile) error {
	if profile.ID.String() == "" {
		return fmt.Errorf("profile.ID is required")
	}

	m.Lock()
	m.store[profile.ID] = profile
	m.Unlock()
	return nil
}

// PeernameID gives the ID for a given peername
func (m *MemStore) PeernameID(peername string) (ID, error) {
	m.Lock()
	defer m.Unlock()

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
func (m *MemStore) PeerProfile(id peer.ID) (*Profile, error) {
	m.Lock()
	defer m.Unlock()

	// str := fmt.Sprintf("/ipfs/%s", id.Pretty())

	for _, profile := range m.store {
		for _, pid := range profile.PeerIDs {
			if id == pid {
				return profile, nil
			}
		}
	}

	return nil, ErrNotFound
}

// PeerIDs gives the peer.IDs list for a given peername
func (m *MemStore) PeerIDs(id ID) ([]peer.ID, error) {
	m.Lock()
	defer m.Unlock()

	for proid, profile := range m.store {
		if id == proid {
			return profile.PeerIDs, nil
		}
	}

	return nil, ErrNotFound
}

// List hands the full list of peers back
func (m *MemStore) List() (map[ID]*Profile, error) {
	m.Lock()
	defer m.Unlock()

	res := map[ID]*Profile{}
	for id, p := range m.store {
		res[id] = p
	}
	return res, nil
}

// GetProfile give's peer info from the store for a given peer.ID
func (m *MemStore) GetProfile(id ID) (*Profile, error) {
	m.Lock()
	defer m.Unlock()

	if m.store[id] == nil {
		return nil, ErrNotFound
	}
	return m.store[id], nil
}

// DeleteProfile removes a peer from this store
func (m *MemStore) DeleteProfile(id ID) error {
	m.Lock()
	delete(m.store, id)
	m.Unlock()

	return nil
}
