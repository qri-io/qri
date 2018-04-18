package fsrepo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/repo/profile"

	"gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

// ErrNotFound is for when a qri profile isn't found
var ErrNotFound = fmt.Errorf("Not Found")

// ProfileStore is an on-disk json file implementation of the
// repo.Peers interface
type ProfileStore struct {
	sync.Mutex
	basepath
}

// NewProfileStore allocates a ProfileStore
func NewProfileStore(bp basepath) ProfileStore {
	return ProfileStore{
		basepath: bp,
	}
}

// PutProfile adds a peer to the store
func (r ProfileStore) PutProfile(p *profile.Profile) error {
	if p.ID.String() == "" {
		return fmt.Errorf("profile ID is required")
	}

	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return err
	}
	ps[p.ID] = p
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

	for proid, profile := range ps {
		if id == proid {
			return profile.PeerIDs(), nil
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
	}
	return ps, err
}

// PeernameID gives the profile.ID for a given peername
func (r ProfileStore) PeernameID(peername string) (profile.ID, error) {
	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return "", err
	}

	for _, profile := range ps {
		if profile.Peername == peername {
			return profile.ID, nil
		}
	}
	return "", datastore.ErrNotFound
}

// GetProfile fetches a profile from the store
func (r ProfileStore) GetProfile(id profile.ID) (*profile.Profile, error) {
	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return nil, err
	}

	for proid, d := range ps {
		if id == proid {
			return d, nil
		}
	}

	return nil, datastore.ErrNotFound
}

// PeerProfile gives the profile that corresponds with a given peer.ID
func (r ProfileStore) PeerProfile(id peer.ID) (*profile.Profile, error) {
	r.Lock()
	defer r.Unlock()

	ps, err := r.profiles()
	if err != nil {
		return nil, err
	}

	for _, profile := range ps {
		if _, ok := profile.Addresses[id.Pretty()]; ok {
			return profile, nil
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
	delete(ps, id)
	return r.saveFile(ps, FilePeers)
}

func (r ProfileStore) saveFile(ps map[profile.ID]*profile.Profile, f File) error {
	pss := map[string]*profile.Profile{}
	for _, p := range ps {
		pss[p.ID.String()] = p
	}

	data, err := json.Marshal(pss)
	if err != nil {
		log.Debug(err.Error())
		return err
	}
	return ioutil.WriteFile(r.filepath(f), data, os.ModePerm)
}

func (r *ProfileStore) profiles() (map[profile.ID]*profile.Profile, error) {
	// pss := map[string]*profile.Profile{}
	ps := map[profile.ID]*profile.Profile{}
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
	// for _, p := range pss {
	// 	ps[p.ID] = p
	// }
	return ps, nil
}
