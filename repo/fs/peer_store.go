package fsrepo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/doggos"
	"github.com/qri-io/qri/repo/profile"

	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// PeerStore is an on-disk json file implementation of the
// repo.Peers interface
type PeerStore struct {
	basepath
}

// PutPeer adds a peer to the store
func (r PeerStore) PutPeer(id peer.ID, p *profile.Profile) error {
	ps, err := r.peers()
	if err != nil {
		return err
	}
	if p.Peername == "" {
		p.Peername = doggos.DoggoNick(id.Pretty())
	}
	ps[id.Pretty()] = p
	return r.saveFile(ps, FilePeers)
}

// List hands back the list of peers
func (r PeerStore) List() (map[string]*profile.Profile, error) {
	ps, err := r.peers()
	if err != nil && err.Error() == "EOF" {
		return map[string]*profile.Profile{}, nil
	}
	return ps, err
}

// GetID gives the peer.ID for a given peername
func (r PeerStore) GetID(peername string) (peer.ID, error) {
	ps, err := r.peers()
	if err != nil {
		return "", err
	}

	for _, profile := range ps {
		if profile.Peername == peername {
			return profile.PeerID()
		}
	}
	return "", datastore.ErrNotFound
}

// GetPeer fetches a peer from the store
func (r PeerStore) GetPeer(id peer.ID) (*profile.Profile, error) {
	ps, err := r.peers()
	if err != nil {
		return nil, err
	}

	ids := id.Pretty()
	for p, d := range ps {
		if ids == p {
			return d, nil
		}
	}

	return nil, datastore.ErrNotFound
}

// IPFSPeerID gives the IPFS peer.ID for a given peername
func (r PeerStore) IPFSPeerID(peername string) (peer.ID, error) {
	ps, err := r.peers()
	if err != nil {
		return "", err
	}

	for id, profile := range ps {
		if profile.Peername == peername {
			if ipfspid, err := profile.IPFSPeerID(); err == nil {
				return ipfspid, nil
			}
			return peer.ID(id), nil
		}
	}

	return "", datastore.ErrNotFound
}

// DeletePeer removes a peer from the store
func (r PeerStore) DeletePeer(id peer.ID) error {
	ps, err := r.peers()
	if err != nil {
		return err
	}
	delete(ps, id.Pretty())
	return r.saveFile(ps, FilePeers)
}

// Query fetches a set of peers from the store according to given query
// parameters
func (r PeerStore) Query(q query.Query) (query.Results, error) {
	ps, err := r.peers()
	if err != nil {
		return nil, err
	}

	re := make([]query.Entry, 0, len(ps))
	for id, peer := range ps {
		if peer.Peername == "" {
			peer.Peername = doggos.DoggoNick(id)
		}
		re = append(re, query.Entry{Key: id, Value: peer})
	}
	res := query.ResultsWithEntries(q, re)
	res = query.NaiveQueryApply(q, res)
	return res, nil
}

func (r *PeerStore) peers() (map[string]*profile.Profile, error) {
	ps := map[string]*profile.Profile{}
	data, err := ioutil.ReadFile(r.filepath(FilePeers))
	if err != nil {
		if os.IsNotExist(err) {
			return ps, nil
		}
		return ps, fmt.Errorf("error loading peers: %s", err.Error())
	}

	if err := json.Unmarshal(data, &ps); err != nil {
		return ps, fmt.Errorf("error unmarshaling peers: %s", err.Error())
	}
	return ps, nil
}
