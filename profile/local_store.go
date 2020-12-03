package profile

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/config"
	"github.com/theckman/go-flock"
)

// LocalStore is an on-disk json file implementation of the
// repo.Peers interface
type LocalStore struct {
	sync.Mutex
	owner    *Profile
	filename string
	flock    *flock.Flock
}

// NewLocalStore allocates a LocalStore
func NewLocalStore(filename string, owner *Profile) (Store, error) {
	if err := owner.ValidOwnerProfile(); err != nil {
		return nil, err
	}

	return &LocalStore{
		owner:    owner,
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
	return nil
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
