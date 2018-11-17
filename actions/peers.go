package actions

import (
	"fmt"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
)

// ListPeers lists Peers on the qri network
func ListPeers(node *p2p.QriNode, limit, offset int, onlineOnly bool) ([]*config.ProfilePod, error) {
	r := node.Repo
	user, err := r.Profile()
	if err != nil {
		return nil, err
	}

	peers := make([]*config.ProfilePod, limit)
	online := []*config.ProfilePod{}
	if online, err = ConnectedQriProfiles(node, limit); err != nil {
		return nil, err
	}

	if onlineOnly {
		return online, nil
	}

	ps, err := r.Profiles().List()
	if err != nil {
		return nil, fmt.Errorf("error listing peers: %s", err.Error())
	}

	if len(ps) == 0 {
		return []*config.ProfilePod{}, nil
	}

	i := 0
	for _, pro := range ps {
		if i >= limit {
			break
		}
		if pro == nil || pro.ID == user.ID {
			continue
		}

		// TODO - this is dumb use a map
		for _, olp := range online {
			if pro.ID.String() == olp.ID {
				pro.Online = true
			}
		}

		peers[i], err = pro.Encode()
		if err != nil {
			return nil, err
		}

		i++
	}

	return peers, nil
}

// ConnectedQriProfiles gives any currently connected qri profiles to this node
func ConnectedQriProfiles(node *p2p.QriNode, limit int) ([]*config.ProfilePod, error) {
	parsed := []*config.ProfilePod{}
	for _, p := range node.ConnectedQriProfiles() {
		parsed = append(parsed, p)
	}
	return parsed, nil
}
