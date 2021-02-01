package p2p

import (
	"fmt"

	"github.com/qri-io/qri/config"
)

// ListPeers lists Peers on the qri network
func ListPeers(node *QriNode, limit, offset int, onlineOnly bool) ([]*config.ProfilePod, error) {

	r := node.Repo
	user, err := r.Owner()
	if err != nil {
		return nil, err
	}

	peers := make([]*config.ProfilePod, 0, limit)
	connected := node.ConnectedQriProfiles()

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
