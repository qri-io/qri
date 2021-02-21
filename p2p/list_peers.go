package p2p

import (
	"fmt"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/profile"
)

// ListPeers lists Peers on the qri network
// userID is the profile identifier of the user making the request
func ListPeers(node *QriNode, userID profile.ID, offset, limit int, onlineOnly bool) ([]*config.ProfilePod, error) {

	r := node.Repo

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
		if pro == nil || pro.ID == userID {
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
