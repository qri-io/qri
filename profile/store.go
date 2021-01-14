package profile

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/key"
	"github.com/theckman/go-flock"
)

// ErrNotFound is the not found err for the profile package
var ErrNotFound = fmt.Errorf("profile: not found")

// Store is a store of profile information. Stores are owned by a single profile
// that must have an associated private key
type Store interface {
	// Owner is a single profile that represents the current user
	Owner() *Profile
	// SetOwner handles updates to the current user profile at runtime
	SetOwner(own *Profile) error

	// put a profile in the store
	PutProfile(profile *Profile) error
	// get a profile by ID
	GetProfile(id ID) (*Profile, error)
	// remove a profile from the store
	DeleteProfile(id ID) error

	// list all profiles in the store, keyed by ID
	// Deprecated - don't add new callers to this. We should replace this with
	// a better batch accessor
	List() (map[ID]*Profile, error)
	// get a set of peer ids for a given profile ID
	PeerIDs(id ID) ([]peer.ID, error)
	// get a profile for a given peer Identifier
	PeerProfile(id peer.ID) (*Profile, error)
	// get the profile ID for a given peername
	// Depcreated - use GetProfile instead
	PeernameID(peername string) (ID, error)
}

// NewStore creates a profile store from configuration
func NewStore(cfg *config.Config) (Store, error) {
	pro, err := NewProfile(cfg.Profile)
	if err != nil {
		return nil, err
	}

	keyStore, err := key.NewStore(cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Repo == nil {
		return NewMemStore(pro, keyStore)
	}

	switch cfg.Repo.Type {
	case "fs":
		return NewLocalStore(filepath.Join(filepath.Dir(cfg.Path()), "peers.json"), pro, keyStore)
	case "mem":
		return NewMemStore(pro, keyStore)
	default:
		return nil, fmt.Errorf("unknown repo type: %s", cfg.Repo.Type)
	}
}

// MemStore is an in-memory implementation of the profile Store interface
type MemStore struct {
	sync.Mutex
	owner    *Profile
	store    map[ID]*Profile
	keyStore key.Store
}

// NewMemStore allocates a MemStore
func NewMemStore(owner *Profile, ks key.Store) (Store, error) {
	if err := owner.ValidOwnerProfile(); err != nil {
		return nil, err
	}

	return &MemStore{
		owner: owner,
		store: map[ID]*Profile{
			owner.ID: owner,
		},
		keyStore: ks,
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
func (m *MemStore) PutProfile(p *Profile) error {
	if p.ID.String() == "" {
		return fmt.Errorf("profile.ID is required")
	}

	m.Lock()
	m.store[p.ID] = p
	m.Unlock()

	if p.PubKey != nil {
		if err := m.keyStore.AddPubKey(p.GetKeyID(), p.PubKey); err != nil {
			return err
		}
	}
	if p.PrivKey != nil {
		if err := m.keyStore.AddPrivKey(p.GetKeyID(), p.PrivKey); err != nil {
			return err
		}
	}
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

	pro := m.store[id]
	pro.KeyID = pro.GetKeyID()
	pro.PubKey = m.keyStore.PubKey(pro.GetKeyID())
	pro.PrivKey = m.keyStore.PrivKey(pro.GetKeyID())

	return pro, nil
}

// DeleteProfile removes a peer from this store
func (m *MemStore) DeleteProfile(id ID) error {
	m.Lock()
	delete(m.store, id)
	m.Unlock()

	return nil
}

// LocalStore is an on-disk json file implementation of the
// repo.Peers interface
type LocalStore struct {
	sync.Mutex
	owner    *Profile
	keyStore key.Store
	filename string
	flock    *flock.Flock
}

// NewLocalStore allocates a LocalStore
func NewLocalStore(filename string, owner *Profile, ks key.Store) (Store, error) {
	if err := owner.ValidOwnerProfile(); err != nil {
		return nil, err
	}

	return &LocalStore{
		owner:    owner,
		keyStore: ks,
		filename: filename,
		flock:    flock.NewFlock(lockPath(filename)),
	}, nil
}

func lockPath(filename string) string {
	return fmt.Sprintf("%s.lock", filename)
}

// Owner accesses the current user profile
func (r *LocalStore) Owner() *Profile {
	return r.owner
}

// SetOwner updates the owner profile
func (r *LocalStore) SetOwner(own *Profile) error {
	r.owner = own
	return r.PutProfile(own)
}

// PutProfile adds a peer to the store
func (r *LocalStore) PutProfile(p *Profile) error {
	log.Debugf("put profile: %s", p.ID.String())
	if p.ID.String() == "" {
		return fmt.Errorf("profile ID is required")
	}

	enc, err := p.Encode()
	if err != nil {
		return fmt.Errorf("error encoding profile: %s", err.Error())
	}

	// explicitly remove Online flag
	enc.Online = false

	if p.PubKey != nil {
		if err := r.keyStore.AddPubKey(p.GetKeyID(), p.PubKey); err != nil {
			return err
		}
	}
	if p.PrivKey != nil {
		if err := r.keyStore.AddPrivKey(p.GetKeyID(), p.PrivKey); err != nil {
			return err
		}
	}

	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return err
	}
	ps[p.ID.String()] = enc
	return r.saveFile(ps)
}

// PeerIDs gives the peer.IDs list for a given peername
func (r *LocalStore) PeerIDs(id ID) ([]peer.ID, error) {
	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return nil, err
	}

	ids := id.String()

	for proid, cp := range ps {
		if ids == proid {
			pro := &Profile{}
			if err := pro.Decode(cp); err != nil {
				return nil, err
			}
			return pro.PeerIDs, err
		}
	}

	return nil, ErrNotFound
}

// List hands back the list of peers
func (r *LocalStore) List() (map[ID]*Profile, error) {
	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil && err.Error() == "EOF" {
		return map[ID]*Profile{}, nil
	} else if err != nil {
		return nil, err
	}

	profiles := map[ID]*Profile{}
	for _, cp := range ps {
		pro := &Profile{}
		if err := pro.Decode(cp); err != nil {
			return nil, err
		}
		profiles[pro.ID] = pro
	}

	return profiles, nil
}

// PeernameID gives the ID for a given peername
func (r *LocalStore) PeernameID(peername string) (ID, error) {
	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return "", err
	}

	for id, cp := range ps {
		if cp.Peername == peername {
			return IDB58Decode(id)
		}
	}
	return "", qfs.ErrNotFound
}

// GetProfile fetches a profile from the store
func (r *LocalStore) GetProfile(id ID) (*Profile, error) {
	log.Debugf("get profile: %s", id.String())

	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return nil, err
	}

	ids := id.String()

	for proid, p := range ps {
		if ids == proid {
			pro := &Profile{}
			err := pro.Decode(p)
			pro.KeyID = pro.GetKeyID()
			pro.PubKey = r.keyStore.PubKey(pro.GetKeyID())
			pro.PrivKey = r.keyStore.PrivKey(pro.GetKeyID())
			return pro, err
		}
	}

	return nil, qfs.ErrNotFound
}

// PeerProfile gives the profile that corresponds with a given peer.ID
func (r *LocalStore) PeerProfile(id peer.ID) (*Profile, error) {
	log.Debugf("peerProfile: %s", id.Pretty())

	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return nil, err
	}

	str := fmt.Sprintf("/ipfs/%s", id.Pretty())
	for _, p := range ps {
		for _, id := range p.PeerIDs {
			if id == str {
				pro := &Profile{}
				err := pro.Decode(p)
				return pro, err
			}
		}
	}

	return nil, qfs.ErrNotFound
}

// DeleteProfile removes a profile from the store
func (r *LocalStore) DeleteProfile(id ID) error {
	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return err
	}
	delete(ps, id.String())
	return r.saveFile(ps)
}

func (r *LocalStore) saveFile(ps map[string]*config.ProfilePod) error {

	data, err := json.Marshal(ps)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	log.Debugf("writing profiles: %s", r.filename)
	if err := r.flock.Lock(); err != nil {
		return err
	}
	defer func() {
		r.flock.Unlock()
		log.Debugf("profiles written")
	}()
	return ioutil.WriteFile(r.filename, data, 0644)
}

func (r *LocalStore) profiles() (map[string]*config.ProfilePod, error) {
	log.Debug("reading profiles")

	if err := r.flock.Lock(); err != nil {
		return nil, err
	}
	defer func() {
		log.Debug("profiles read")
		r.flock.Unlock()
	}()

	pp := map[string]*config.ProfilePod{}
	data, err := ioutil.ReadFile(r.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return pp, nil
		}
		log.Debug(err.Error())
		return pp, fmt.Errorf("error loading peers: %s", err.Error())
	}

	if err := json.Unmarshal(data, &pp); err != nil {
		log.Error(err.Error())
		// TODO - this is totally screwed for some reason, so for now when things fail,
		// let's just return an empty list of peers
		return pp, nil
	}
	return pp, nil
}
