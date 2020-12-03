package profile

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/qri/config"
)

// ErrNotFound is the not found err for the profile package
var ErrNotFound = fmt.Errorf("profile: not found")

// Store is a store of profile information
type Store interface {
	// Owner is a single profile that represents the current user
	Owner() *Profile
	// SetOwner handles updates to the current user profile at runtime
	SetOwner(own *Profile) error

	List() (map[ID]*Profile, error)
	PeerIDs(id ID) ([]peer.ID, error)
	PeernameID(peername string) (ID, error)
	PutProfile(profile *Profile) error
	GetProfile(id ID) (*Profile, error)
	PeerProfile(id peer.ID) (*Profile, error)
	DeleteProfile(id ID) error
}

// NewStore creates a profile store from configuration
func NewStore(cfg *config.Config) (Store, error) {
	pro, err := NewProfile(cfg.Profile)
	if err != nil {
		return nil, err
	}

	if cfg.Repo == nil {
		return NewMemStore(pro)
	}

	switch cfg.Repo.Type {
	case "fs":
		return NewLocalStore(filepath.Join(filepath.Dir(cfg.Path()), "peers.json"), pro)
	case "mem":
		return NewMemStore(pro)
	default:
		return nil, fmt.Errorf("unknown repo type: %s", cfg.Repo.Type)
	}
}

// MemStore is an in-memory implementation of the profile Store interface
type MemStore struct {
	sync.Mutex
	owner *Profile
	store map[ID]*Profile
}

// NewMemStore allocates a MemStore
func NewMemStore(owner *Profile) (Store, error) {
	if err := owner.ValidOwnerProfile(); err != nil {
		return nil, err
	}

	return &MemStore{
		owner: owner,
		store: map[ID]*Profile{
			owner.ID: owner,
		},
	}, nil
}

// Owner accesses the current user profile
func (m *MemStore) Owner() *Profile {
	return m.owner
}

// SetOwner updates the owner profile
func (m *MemStore) SetOwner(own *Profile) error {
	m.owner = own
	return nil
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
