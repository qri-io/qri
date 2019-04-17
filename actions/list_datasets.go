package actions

import (
	"fmt"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// ListDatasets lists a peer's datasets
func ListDatasets(node *p2p.QriNode, ds *repo.DatasetRef, term string, limit, offset int, RPC, publishedOnly, showVersions bool) (res []repo.DatasetRef, err error) {

	r := node.Repo
	pro, err := r.Profile()
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error getting profile: %s", err.Error())
	}

	if ds.Peername == "me" {
		ds.Peername = pro.Peername
		ds.ProfileID = pro.ID
	}

	if err := repo.CanonicalizeProfile(r, ds); err != nil {
		return nil, fmt.Errorf("error canonicalizing peer: %s", err.Error())
	}

	if ds.Peername != "" && ds.Peername != pro.Peername {
		if node == nil {
			return nil, fmt.Errorf("cannot list remote datasets without p2p connection")
		}

		var profiles map[profile.ID]*profile.Profile
		profiles, err = r.Profiles().List()
		if err != nil {
			log.Debug(err.Error())
			return nil, fmt.Errorf("error fetching profile: %s", err.Error())
		}

		var pro *profile.Profile
		for _, p := range profiles {
			if ds.ProfileID.String() == p.ID.String() || ds.Peername == p.Peername {
				pro = p
			}
		}
		if err != nil {
			return nil, fmt.Errorf("couldn't find profile: %s", err.Error())
		}
		if pro == nil {
			return nil, fmt.Errorf("profile not found: \"%s\"", ds.Peername)
		}

		if len(pro.PeerIDs) == 0 {
			return nil, fmt.Errorf("couldn't find a peer address for profile: %s", pro.ID)
		}

		res, err = node.RequestDatasetsList(pro.PeerIDs[0], p2p.DatasetsListParams{
			Term:   term,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("error requesting dataset list: %s", err.Error())
		}
		// TODO - for now we're removing schemas b/c they don't serialize properly over RPC
		if RPC {
			for _, rep := range res {
				if rep.Dataset.Structure != nil {
					rep.Dataset.Structure.Schema = nil
				}
			}
		}
		return
	}

	return base.ListDatasets(node.Repo, term, limit, offset, RPC, publishedOnly, showVersions)
}
