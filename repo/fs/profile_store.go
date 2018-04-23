package fsrepo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/repo/profile"
	"github.com/theckman/go-flock"

	"gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

// ErrNotFound is for when a qri profile isn't found
var ErrNotFound = fmt.Errorf("Not Found")

// ProfileStore is an on-disk json file implementation of the
// repo.Peers interface
type ProfileStore struct {
	sync.Mutex
	basepath
	flock *flock.Flock
}

// NewProfileStore allocates a ProfileStore
func NewProfileStore(bp basepath) ProfileStore {
	return ProfileStore{
		basepath: bp,
		flock:    flock.NewFlock(bp.filepath(FilePeers) + ".lock"),
	}
}

// PutProfile adds a peer to the store
func (r ProfileStore) PutProfile(p *profile.Profile) error {
	log.Debugf("put profile: %s", p.ID.String())
	if p.ID.String() == "" {
		return fmt.Errorf("profile ID is required")
	}

	enc, err := p.Encode()
	if err != nil {
		return fmt.Errorf("error encoding profile: %s", err.Error())
	}

	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return err
	}
	ps[p.ID.String()] = enc
	return r.saveFile(ps, FilePeers)
}

// PeerIDs gives the peer.IDs list for a given peername
func (r ProfileStore) PeerIDs(id profile.ID) ([]peer.ID, error) {
	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return nil, err
	}

	ids := id.String()

	for proid, cp := range ps {
		if ids == proid {
			pro := &profile.Profile{}
			if err := pro.Decode(cp); err != nil {
				return nil, err
			}
			return pro.PeerIDs(), err
		}
	}

	return nil, ErrNotFound
}

// List hands back the list of peers
func (r ProfileStore) List() (map[profile.ID]*profile.Profile, error) {
	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil && err.Error() == "EOF" {
		return map[profile.ID]*profile.Profile{}, nil
	} else if err != nil {
		return nil, err
	}

	profiles := map[profile.ID]*profile.Profile{}
	for _, cp := range ps {
		pro := &profile.Profile{}
		if err := pro.Decode(cp); err != nil {
			return nil, err
		}
		profiles[pro.ID] = pro
	}

	return profiles, nil
}

// PeernameID gives the profile.ID for a given peername
func (r ProfileStore) PeernameID(peername string) (profile.ID, error) {
	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return "", err
	}

	for id, cp := range ps {
		if cp.Peername == peername {
			return profile.IDB58Decode(id)
		}
	}
	return "", datastore.ErrNotFound
}

// GetProfile fetches a profile from the store
func (r ProfileStore) GetProfile(id profile.ID) (*profile.Profile, error) {
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
			pro := &profile.Profile{}
			err := pro.Decode(p)
			return pro, err
		}
	}

	return nil, datastore.ErrNotFound
}

// PeerProfile gives the profile that corresponds with a given peer.ID
func (r ProfileStore) PeerProfile(id peer.ID) (*profile.Profile, error) {
	log.Debugf("peerProfile: %s", id.String())

	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return nil, err
	}

	for _, p := range ps {
		if _, ok := p.Addresses[id.String()]; ok {
			pro := &profile.Profile{}
			err := pro.Decode(p)
			return pro, err
		}
	}

	return nil, datastore.ErrNotFound
}

// DeleteProfile removes a profile from the store
func (r ProfileStore) DeleteProfile(id profile.ID) error {
	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return err
	}
	delete(ps, id.String())
	return r.saveFile(ps, FilePeers)
}

func (r ProfileStore) saveFile(ps map[string]*profile.CodingProfile, f File) error {

	data, err := json.Marshal(ps)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	log.Debugf("writing profiles: %s", r.filepath(f))
	if err := r.flock.Lock(); err != nil {
		return err
	}
	defer func() {
		r.flock.Unlock()
		log.Debugf("profiles written")
	}()
	return ioutil.WriteFile(r.filepath(f), data, os.ModePerm)
}

func (r *ProfileStore) profiles() (map[string]*profile.CodingProfile, error) {
	log.Debug("reading profiles")

	if err := r.flock.Lock(); err != nil {
		return nil, err
	}
	defer func() {
		log.Debug("profiles read")
		r.flock.Unlock()
	}()

	ps := map[string]*profile.CodingProfile{}
	data, err := ioutil.ReadFile(r.filepath(FilePeers))
	if err != nil {
		if os.IsNotExist(err) {
			return ps, nil
		}
		log.Debug(err.Error())
		return ps, fmt.Errorf("error loading peers: %s", err.Error())
	}

	if err := json.Unmarshal(data, &ps); err != nil {
		log.Error(err.Error())
		// TODO - this is totally screwed for some reason, so for now when things fail,
		// let's just return an empty list of peers
		return ps, nil
		// return ps, fmt.Errorf("error unmarshaling peers: %s", err.Error())
	}
	return ps, nil
}
