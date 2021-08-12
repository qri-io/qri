package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gofrs/flock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/config"
	qerr "github.com/qri-io/qri/errors"
)

var (
	// ErrNotFound is the not found err for the profile package
	ErrNotFound = fmt.Errorf("profile: not found")
	// ErrAmbiguousUsername occurs when more than one username is the same in a
	// context that requires exactly one user. More information is needed to
	// disambiguate which username is correct
	ErrAmbiguousUsername = fmt.Errorf("ambiguous username")
)

// Store is a store of profile information. Stores are owned by a single profile
// that must have an associated private key
type Store interface {
	// Owner is a single profile that represents the current user
	Owner(ctx context.Context) *Profile
	// SetOwner handles updates to the current user profile at runtime
	SetOwner(ctx context.Context, own *Profile) error
	// Active is the active profile that represents the current user
	Active(ctx context.Context) *Profile

	// put a profile in the store
	PutProfile(ctx context.Context, profile *Profile) error
	// get a profile by ID
	GetProfile(ctx context.Context, id ID) (*Profile, error)
	// remove a profile from the store
	DeleteProfile(ctx context.Context, id ID) error

	// get all profiles who's .Peername field matches a given username. It's
	// possible to have multiple profiles with the same username
	ProfilesForUsername(ctx context.Context, username string) ([]*Profile, error)
	// list all profiles in the store, keyed by ID
	// Deprecated - don't add new callers to this. We should replace this with
	// a better batch accessor
	List(ctx context.Context) (map[ID]*Profile, error)
	// get a set of peer ids for a given profile ID
	PeerIDs(ctx context.Context, id ID) ([]peer.ID, error)
	// get a profile for a given peer Identifier
	PeerProfile(ctx context.Context, id peer.ID) (*Profile, error)
	// get the profile ID for a given peername
	// Depcreated - use GetProfile instead
	PeernameID(ctx context.Context, peername string) (ID, error)
}

// NewStore creates a profile store from configuration
func NewStore(ctx context.Context, cfg *config.Config, keyStore key.Store) (Store, error) {
	pro, err := NewProfile(cfg.Profile)
	if err != nil {
		return nil, err
	}

	// Don't create a localstore with the empty path, this will use the current directory
	if cfg.Repo.Type == "fs" && cfg.Path() == "" {
		return nil, fmt.Errorf("new Profile.FilesystemStore requires non-empty path")
	}

	if cfg.Repo == nil {
		return NewMemStore(ctx, pro, keyStore)
	}

	switch cfg.Repo.Type {
	case "fs":
		return NewLocalStore(ctx, filepath.Join(filepath.Dir(cfg.Path()), "peers.json"), pro, keyStore)
	case "mem":
		return NewMemStore(ctx, pro, keyStore)
	default:
		return nil, fmt.Errorf("unknown repo type: %s", cfg.Repo.Type)
	}
}

// ResolveUsername finds a single profile for a given username from a store of
// usernames. Errors if the store contains more than one user with the given
// username
func ResolveUsername(ctx context.Context, s Store, username string) (*Profile, error) {
	pros, err := s.ProfilesForUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	if len(pros) > 1 {
		return nil, newAmbiguousUsernamesError(pros)
	} else if len(pros) == 0 {
		return nil, ErrNotFound
	}

	return pros[0], nil
}

// NewAmbiguousUsernamesError creates a qri error that describes how to choose
// the right user
// TODO(b5): this message doesn't describe a fix... because we don't have a good
// one yet. We need to modify dsref parsing to deal with username disambiguation
func newAmbiguousUsernamesError(pros []*Profile) error {
	msg := ""
	if len(pros) > 0 {
		descriptions := make([]string, len(pros), len(pros))
		for i, p := range pros {
			descriptions[i] = fmt.Sprintf("%s\t%s", p.ID, p.Email)
		}
		msg = fmt.Sprintf("multiple profiles exist for the username %q.\nprofileID\temail\n%s", pros[0].Peername, strings.Join(descriptions, "\n"))
	}
	return qerr.New(ErrAmbiguousUsername, msg)
}

// MemStore is an in-memory implementation of the profile Store interface
type MemStore struct {
	sync.Mutex
	owner    *Profile
	store    map[ID]*Profile
	keyStore key.Store
}

// NewMemStore allocates a MemStore
func NewMemStore(ctx context.Context, owner *Profile, ks key.Store) (Store, error) {
	if err := owner.ValidOwnerProfile(); err != nil {
		return nil, err
	}

	if err := ks.AddPrivKey(ctx, owner.GetKeyID(), owner.PrivKey); err != nil {
		return nil, err
	}
	if err := ks.AddPubKey(ctx, owner.GetKeyID(), owner.PrivKey.GetPublic()); err != nil {
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

// Owner accesses the current owner user profile
func (m *MemStore) Owner(ctx context.Context) *Profile {
	// TODO(b5): this should return a copy
	return m.owner
}

// SetOwner updates the owner profile
func (m *MemStore) SetOwner(ctx context.Context, own *Profile) error {
	m.owner = own
	return nil
}

// Active is the curernt active profile
func (m *MemStore) Active(ctx context.Context) *Profile {
	return m.Owner(ctx)
}

// PutProfile adds a peer to this store
func (m *MemStore) PutProfile(ctx context.Context, p *Profile) error {
	if p.ID.Empty() {
		return fmt.Errorf("profile.ID is required")
	}

	m.Lock()
	m.store[p.ID] = p
	m.Unlock()

	if p.PubKey != nil {
		if err := m.keyStore.AddPubKey(ctx, p.GetKeyID(), p.PubKey); err != nil {
			return err
		}
	}
	if p.PrivKey != nil {
		if err := m.keyStore.AddPrivKey(ctx, p.GetKeyID(), p.PrivKey); err != nil {
			return err
		}
	}
	return nil
}

// PeernameID gives the ID for a given peername
func (m *MemStore) PeernameID(ctx context.Context, peername string) (ID, error) {
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
func (m *MemStore) PeerProfile(ctx context.Context, id peer.ID) (*Profile, error) {
	m.Lock()
	defer m.Unlock()

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
func (m *MemStore) PeerIDs(ctx context.Context, id ID) ([]peer.ID, error) {
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
func (m *MemStore) List(ctx context.Context) (map[ID]*Profile, error) {
	m.Lock()
	defer m.Unlock()

	res := map[ID]*Profile{}
	for id, p := range m.store {
		res[id] = p
	}
	return res, nil
}

// GetProfile give's peer info from the store for a given peer.ID
func (m *MemStore) GetProfile(ctx context.Context, id ID) (*Profile, error) {
	m.Lock()
	defer m.Unlock()

	if m.store[id] == nil {
		return nil, ErrNotFound
	}

	pro := m.store[id]
	pro.KeyID = pro.GetKeyID()
	pro.PubKey = m.keyStore.PubKey(ctx, pro.GetKeyID())
	pro.PrivKey = m.keyStore.PrivKey(ctx, pro.GetKeyID())

	return pro, nil
}

// ProfilesForUsername fetches all profile that match a username (Peername)
func (m *MemStore) ProfilesForUsername(ctx context.Context, username string) ([]*Profile, error) {
	m.Lock()
	defer m.Unlock()

	var res []*Profile
	for _, pro := range m.store {
		if pro.Peername == username {
			res = append(res, pro)
		}
	}

	return res, nil
}

// DeleteProfile removes a peer from this store
func (m *MemStore) DeleteProfile(ctx context.Context, id ID) error {
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
func NewLocalStore(ctx context.Context, filename string, owner *Profile, ks key.Store) (Store, error) {
	if err := owner.ValidOwnerProfile(); err != nil {
		return nil, err
	}

	if err := ks.AddPrivKey(ctx, owner.GetKeyID(), owner.PrivKey); err != nil {
		return nil, err
	}

	s := &LocalStore{
		owner:    owner,
		keyStore: ks,
		filename: filename,
		flock:    flock.New(lockPath(filename)),
	}

	err := s.PutProfile(ctx, owner)
	return s, err
}

func lockPath(filename string) string {
	return fmt.Sprintf("%s.lock", filename)
}

// Owner accesses the current owner user profile
func (r *LocalStore) Owner(ctx context.Context) *Profile {
	// TODO(b5): this should return a copy
	return r.owner
}

// SetOwner updates the owner profile
func (r *LocalStore) SetOwner(ctx context.Context, own *Profile) error {
	r.owner = own
	return r.PutProfile(ctx, own)
}

// Active is the curernt active profile
func (r *LocalStore) Active(ctx context.Context) *Profile {
	return r.Owner(ctx)
}

// PutProfile adds a peer to the store
func (r *LocalStore) PutProfile(ctx context.Context, p *Profile) error {
	log.Debugf("put profile: %s", p.ID.Encode())
	if p.ID.Empty() {
		return fmt.Errorf("profile ID is required")
	}

	enc, err := p.Encode()
	if err != nil {
		return fmt.Errorf("error encoding profile: %s", err.Error())
	}

	// explicitly remove Online flag
	enc.Online = false

	if p.PubKey != nil {
		if err := r.keyStore.AddPubKey(ctx, p.GetKeyID(), p.PubKey); err != nil {
			return err
		}
	}
	if p.PrivKey != nil {
		if err := r.keyStore.AddPrivKey(ctx, p.GetKeyID(), p.PrivKey); err != nil {
			return err
		}
	}

	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return err
	}
	ps[p.ID.Encode()] = enc
	return r.saveFile(ps)
}

// PeerIDs gives the peer.IDs list for a given peername
func (r *LocalStore) PeerIDs(ctx context.Context, id ID) ([]peer.ID, error) {
	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return nil, err
	}

	ids := id.Encode()

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
func (r *LocalStore) List(ctx context.Context) (map[ID]*Profile, error) {
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
func (r *LocalStore) PeernameID(ctx context.Context, peername string) (ID, error) {
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
	return "", ErrNotFound
}

// GetProfile fetches a profile from the store
func (r *LocalStore) GetProfile(ctx context.Context, id ID) (*Profile, error) {
	log.Debugf("get profile: %s", id.Encode())

	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return nil, err
	}

	ids := id.Encode()

	for proid, p := range ps {
		if ids == proid {
			pro := &Profile{}
			err := pro.Decode(p)
			pro.KeyID = pro.GetKeyID()
			pro.PubKey = r.keyStore.PubKey(ctx, pro.GetKeyID())
			pro.PrivKey = r.keyStore.PrivKey(ctx, pro.GetKeyID())
			return pro, err
		}
	}

	return nil, ErrNotFound
}

// ProfilesForUsername fetches all profile that match a username (Peername)
func (r *LocalStore) ProfilesForUsername(ctx context.Context, username string) ([]*Profile, error) {
	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return nil, err
	}

	var res []*Profile

	for id, p := range ps {
		if p.Peername == username {
			pro := &Profile{}
			if err := pro.Decode(p); err != nil {
				log.Debugw("decoding LocalStore profile", "id", id, "err", err)
				continue
			}
			pro.KeyID = pro.GetKeyID()
			pro.PubKey = r.keyStore.PubKey(ctx, pro.GetKeyID())
			pro.PrivKey = r.keyStore.PrivKey(ctx, pro.GetKeyID())
			res = append(res, pro)
		}
	}

	return res, nil
}

// PeerProfile gives the profile that corresponds with a given peer.ID
func (r *LocalStore) PeerProfile(ctx context.Context, id peer.ID) (*Profile, error) {
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

	return nil, ErrNotFound
}

// DeleteProfile removes a profile from the store
func (r *LocalStore) DeleteProfile(ctx context.Context, id ID) error {
	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return err
	}
	delete(ps, id.Encode())
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
