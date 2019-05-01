package actions

import (
	"fmt"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo/profile"
)

// ListPeers lists Peers on the qri network
func ListPeers(node *p2p.QriNode, limit, offset int, onlineOnly bool) ([]*config.ProfilePod, error) {

	r := node.Repo
	user, err := r.Profile()
	if err != nil {
		return nil, err
	}

	peers := make([]*config.ProfilePod, 0, limit)
	connected, err := ConnectedQriProfiles(node)
	if err != nil {
		return nil, err
	}

	if onlineOnly {
		for _, p := range connected {
			peers = append(peers, p)
		}
		return peers, nil
	}

	ps, err := r.Profiles().List()
	if err != nil {
		return nil, fmt.Errorf("error listing peers: %s", err.Error())
	}

	if len(ps) == 0 || offset >= len(ps) {
		return []*config.ProfilePod{}, nil
	}

	for _, pro := range ps {
		if offset > 0 {
			offset--
			continue
		}
		if len(peers) >= limit {
			break
		}
		if pro == nil || pro.ID == user.ID {
			continue
		}

		if _, ok := connected[pro.ID]; ok {
			pro.Online = true
		}

		p, err := pro.Encode()
		if err != nil {
			return nil, err
		}
		peers = append(peers, p)
	}

	return peers, nil
}

// ConnectedQriProfiles returns a map from ProfileIDs to profiles for each connected node
func ConnectedQriProfiles(node *p2p.QriNode) (map[profile.ID]*config.ProfilePod, error) {
	return node.ConnectedQriProfiles(), nil
}
