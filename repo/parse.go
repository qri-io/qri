package repo

import (
	"fmt"
	"strings"

	"github.com/mr-tron/base58/base58"
	"github.com/multiformats/go-multihash"
	reporef "github.com/qri-io/qri/repo/ref"
	"github.com/qri-io/qri/repo/profile"
)

// MustParseDatasetRef panics if the reference is invalid. Useful for testing
func MustParseDatasetRef(refstr string) reporef.DatasetRef {
	ref, err := ParseDatasetRef(refstr)
	if err != nil {
		panic(err)
	}
	return ref
}

// ParseDatasetRef decodes a dataset reference from a string value
// It’s possible to refer to a dataset in a number of ways.
// The full definition of a dataset reference is as follows:
//     dataset_reference = peer_name/dataset_name@peer_id/network/hash
//
// we swap in defaults as follows, all of which are represented as
// empty strings:
//     network - defaults to /ipfs/
//     hash - tip of version history (latest known commit)
//
// these defaults are currently enforced by convention.
// TODO - make Dataset Ref parsing the responisiblity of the Repo
// interface, replacing empty strings with actual defaults
//
// dataset names & hashes are disambiguated by checking if the input
// parses to a valid multihash after base58 decoding.
// through defaults & base58 checking the following should all parse:
//     peer_name/dataset_name
//     /network/hash
//     peername
//     peer_id
//     @peer_id
//     @peer_id/network/hash
//
// see tests for more exmples
//
// TODO - add validation that prevents peernames from being
// valid base58 multihashes and makes sure hashes are actually valid base58 multihashes
// TODO - figure out how IPFS CID's play into this
func ParseDatasetRef(ref string) (reporef.DatasetRef, error) {
	if ref == "" {
		return reporef.DatasetRef{}, ErrEmptyRef
	}

	var (
		// nameRefString string
		dsr = reporef.DatasetRef{}
		err error
	)

	// if there is an @ symbol, we are dealing with a reporef.DatasetRef
	// with an identifier
	atIndex := strings.Index(ref, "@")

	if atIndex != -1 {

		dsr.Peername, dsr.Name = parseAlias(ref[:atIndex])
		dsr.ProfileID, dsr.Path, err = parseIdentifiers(ref[atIndex+1:])

	} else {

		var peername, datasetname, pid bool
		toks := strings.Split(ref, "/")

		for i, tok := range toks {
			if isBase58Multihash(tok) {
				// first hash we encounter is a peerID
				if !pid {
					dsr.ProfileID, _ = profile.IDB58Decode(tok)
					pid = true
					continue
				}

				if !isBase58Multihash(toks[i-1]) {
					dsr.Path = fmt.Sprintf("/%s/%s", toks[i-1], strings.Join(toks[i:], "/"))
				} else {
					dsr.Path = fmt.Sprintf("/ipfs/%s", strings.Join(toks[i:], "/"))
				}
				break
			}

			if !peername {
				dsr.Peername = tok
				peername = true
				continue
			}

			if !datasetname {
				dsr.Name = tok
				datasetname = true
				continue
			}

			dsr.Path = strings.Join(toks[i:], "/")
			break
		}
	}

	if dsr.ProfileID == "" && dsr.Peername == "" && dsr.Name == "" && dsr.Path == "" {
		err = fmt.Errorf("malformed reporef.DatasetRef string: %s", ref)
		return dsr, err
	}

	// if dsr.ProfileID != "" {
	// 	if !isBase58Multihash(dsr.ProfileID) {
	// 		err = fmt.Errorf("invalid ProfileID: '%s'", dsr.ProfileID)
	// 		return dsr, err
	// 	}
	// }

	return dsr, err
}

func parseAlias(alias string) (peer, dataset string) {
	for i, tok := range strings.Split(alias, "/") {
		switch i {
		case 0:
			peer = tok
		case 1:
			dataset = tok
		}
	}
	return
}

func parseIdentifiers(ids string) (profileID profile.ID, path string, err error) {

	toks := strings.Split(ids, "/")
	switch len(toks) {
	case 0:
		err = fmt.Errorf("malformed reporef.DatasetRef identifier: %s", ids)
	case 1:
		if toks[0] != "" {
			profileID, err = profile.IDB58Decode(toks[0])
			// if !isBase58Multihash(toks[0]) {
			// 	err = fmt.Errorf("'%s' is not a base58 multihash", ids)
			// }

			return
		}
	case 2:
		if pid, e := profile.IDB58Decode(toks[0]); e == nil {
			profileID = pid
		}

		if isBase58Multihash(toks[0]) && isBase58Multihash(toks[1]) {
			toks[1] = fmt.Sprintf("/ipfs/%s", toks[1])
		}

		path = toks[1]
	default:
		if pid, e := profile.IDB58Decode(toks[0]); e == nil {
			profileID = pid
		}

		path = fmt.Sprintf("/%s/%s", toks[1], toks[2])
		return
	}

	return
}

// TODO - this could be more robust?
func stripProtocol(ref string) string {
	if strings.HasPrefix(ref, "/ipfs/") {
		return ref[len("/ipfs/"):]
	}
	return ref
}

func isBase58Multihash(hash string) bool {
	data, err := base58.Decode(hash)
	if err != nil {
		return false
	}
	if _, err := multihash.Decode(data); err != nil {
		return false
	}

	return true
}
