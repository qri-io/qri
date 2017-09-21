package fs_repo

import (
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/qri-io/qri/repo/profile"
	"io/ioutil"
	"os"
)

type PeerStore struct {
	basepath
}

func (r PeerStore) PutPeer(id peer.ID, p *profile.Profile) error {
	ps, err := r.peers()
	if err != nil {
		return err
	}
	ps[id] = p
	return r.saveFile(ps, FilePeers)
}

func (r PeerStore) GetPeer(id peer.ID) (*profile.Profile, error) {
	ps, err := r.peers()
	if err != nil {
		return nil, err
	}
	for p, d := range ps {
		if id == p {
			return d, nil
		}
	}

	return nil, datastore.ErrNotFound
}

func (r PeerStore) DeletePeer(id peer.ID) error {
	ps, err := r.peers()
	if err != nil {
		return err
	}
	delete(ps, id)
	return r.saveFile(ps, FilePeers)
}

func (r PeerStore) Query(q query.Query) (query.Results, error) {
	ps, err := r.peers()
	if err != nil {
		return nil, err
	}

	re := make([]query.Entry, 0, len(ps))
	for id, peer := range ps {
		re = append(re, query.Entry{Key: id.String(), Value: peer})
	}
	res := query.ResultsWithEntries(q, re)
	res = query.NaiveQueryApply(q, res)
	return res, nil
}

func (r *PeerStore) peers() (map[peer.ID]*profile.Profile, error) {
	ps := map[peer.ID]*profile.Profile{}
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
