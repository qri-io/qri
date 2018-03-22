package repo

import (
	"fmt"
	"strings"

	"github.com/mr-tron/base58/base58"
	"github.com/multiformats/go-multihash"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo/profile"
)

// Refstore keeps a collection of dataset references
type Refstore interface {
	// PutRef adds a reference to the store. References must be complete with
	// Peername, Name, and Path specified
	PutRef(ref DatasetRef) error
	GetRef(ref DatasetRef) (DatasetRef, error)
	DeleteRef(ref DatasetRef) error
	References(limit, offset int) ([]DatasetRef, error)
	RefCount() (int, error)
}

// ProfileRef encapsulates a reference to a peer profile
// It's main job is to connect peernames / profile ID's to profiles
type ProfileRef struct {
	Peername  string `json:"peername,omitempty"`
	ProfileID string `json:"profileID,omitempty"`
	// Profile data
	Profile *profile.Profile
}

// String implements the Stringer interface for PeerRef
func (r ProfileRef) String() (s string) {
	s = r.Peername
	if r.ProfileID != "" {
		s += "@" + r.ProfileID
	}
	return
}

// DatasetRef encapsulates a reference to a dataset. This needs to exist to bind
// ways of referring to a dataset to a dataset itself, as datasets can't easily
// contain their own hash information, and names are unique on a per-repository
// basis.
// It's tempting to think this needs to be "bigger", supporting more fields,
// keep in mind that if the information is important at all, it should
// be stored as metadata within the dataset itself.
type DatasetRef struct {
	// Peername of dataset owner
	Peername string `json:"peername,omitempty"`
	// PeerID of dataset owner
	PeerID string `json:"peerID,omitempty"`
	// Unique name reference for this dataset
	Name string `json:"name,omitempty"`
	// Content-addressed path for this dataset
	Path string `json:"path,omitempty"`
	// Dataset is a pointer to the dataset being referenced
	Dataset *dataset.Dataset `json:"dataset,omitempty"`
}

// String implements the Stringer interface for DatasetRef
func (r DatasetRef) String() (s string) {
	s = r.AliasString()
	if r.PeerID != "" || r.Path != "" {
		s += "@"
	}
	if r.PeerID != "" {
		s += r.PeerID
	}
	if r.Path != "" {
		s += r.Path
	}
	return
}

// AliasString returns the alias components of a DatasetRef as a string
func (r DatasetRef) AliasString() (s string) {
	s = r.Peername
	if r.Name != "" {
		s += "/" + r.Name
	}
	return
}

// Match checks returns true if Peername and Name are equal,
// and/or path is equal
func (r DatasetRef) Match(b DatasetRef) bool {
	// fmt.Printf("\nr.Peername: %s b.Peername: %s\n", r.Peername, b.Peername)
	// fmt.Printf("\nr.Name: %s b.Name: %s\n", r.Name, b.Name)
	return (r.Path != "" && b.Path != "" && r.Path == b.Path) || (r.PeerID == b.PeerID || r.Peername == b.Peername) && r.Name == b.Name
}

// Equal returns true only if Peername Name and Path are equal
func (r DatasetRef) Equal(b DatasetRef) bool {
	return r.Peername == b.Peername && r.PeerID == b.PeerID && r.Name == b.Name && r.Path == b.Path
}

// IsPeerRef returns true if only Peername is set
func (r DatasetRef) IsPeerRef() bool {
	return (r.Peername != "" || r.PeerID != "") && r.Name == "" && r.Path == "" && r.Dataset == nil
}

// IsEmpty returns true if none of it's fields are set
func (r DatasetRef) IsEmpty() bool {
	return r.Equal(DatasetRef{})
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
func ParseDatasetRef(ref string) (DatasetRef, error) {
	if ref == "" {
		return DatasetRef{}, fmt.Errorf("cannot parse empty string as dataset reference")
	}

	var (
		// nameRefString string
		dsr = DatasetRef{}
		err error
	)

	// if there is an @ symbol, we are dealing with a DatasetRef
	// with an identifier
	atIndex := strings.Index(ref, "@")

	if atIndex != -1 {

		dsr.Peername, dsr.Name = parseAlias(ref[:atIndex])
		dsr.PeerID, dsr.Path, err = parseIdentifiers(ref[atIndex+1:])

	} else {

		var peername, datasetname, pid bool
		toks := strings.Split(ref, "/")

		for i, tok := range toks {
			if isBase58Multihash(tok) {
				// first hash we encounter is a peerID
				if !pid {
					dsr.PeerID = tok
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

	if dsr.PeerID == "" && dsr.Peername == "" && dsr.Name == "" && dsr.Path == "" {
		err = fmt.Errorf("malformed DatasetRef string: %s", ref)
		return dsr, err
	}

	if dsr.PeerID != "" {
		if !isBase58Multihash(dsr.PeerID) {
			err = fmt.Errorf("invalid PeerID: '%s'", dsr.PeerID)
			return dsr, err
		}
	}

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

func parseIdentifiers(ids string) (peerID, path string, err error) {

	toks := strings.Split(ids, "/")
	switch len(toks) {
	case 0:
		err = fmt.Errorf("malformed DatasetRef identifier: %s", ids)
	case 1:
		if toks[0] != "" {
			peerID = toks[0]
			if !isBase58Multihash(toks[0]) {
				err = fmt.Errorf("'%s' is not a base58 multihash", ids)
			}
			return
		}
	case 2:
		peerID = toks[0]
		if isBase58Multihash(toks[0]) && isBase58Multihash(toks[1]) {
			toks[1] = fmt.Sprintf("/ipfs/%s", toks[1])
		}

		path = toks[1]
	default:
		peerID = toks[0]
		path = fmt.Sprintf("/%s/%s", toks[1], toks[2])
		// path = fmt.Sprintf("/%s", strings.Join(toks[1:], "/"))
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

// CanonicalizeDatasetRef uses a repo to turn any local aliases into known
// canonical peername for a dataset and populates a missing path
// if the repo has path information for a peername/name combo
// if we provide any other shortcuts for names other than "me"
// in the future, it should be handled here.
func CanonicalizeDatasetRef(r Repo, ref *DatasetRef) error {
	// when operating over RPC there's a good chance we won't have a repo, in that
	// case we're going to have to rely on the other end of the wire to do canonicalization
	// TODO - think carefully about placement of reference parsing, possibly moving
	// this into core functions.
	if r == nil {
		return nil
	}

	if err := CanonicalizePeer(r, ref); err != nil {
		return err
	}

	if ref.Path != "" && ref.PeerID != "" && ref.Name != "" && ref.Peername != "" {
		return nil
	}

	got, err := r.GetRef(*ref)
	if err == nil {
		if ref.Path == "" {
			ref.Path = got.Path
		}
		if ref.PeerID == "" {
			ref.PeerID = got.PeerID
		}
		if ref.Name == "" {
			ref.Name = got.Name
		}
		if ref.Peername == "" {
			ref.Peername = got.Peername
		}
		if ref.Path != got.Path || ref.PeerID != got.PeerID || ref.Name != got.Name || ref.Peername != got.Peername {
			return fmt.Errorf("Given datasetRef %s does not match datasetRef on file: %s", ref.String(), got.String())
		}
	}

	return nil
}

// CanonicalizePeer populates dataset DatasetRef PeerID and Peername properties,
// changing aliases to known names, and adding PeerID from a peerstore
func CanonicalizePeer(r Repo, ref *DatasetRef) error {
	if ref.Peername == "" && ref.PeerID == "" {
		return nil
	}

	p, err := r.Profile()
	if err != nil {
		return err
	}

	if ref.Peername == "me" || ref.Peername == p.Peername || ref.PeerID == p.ID {
		if ref.Peername == "me" {
			ref.PeerID = p.ID
			ref.Peername = p.Peername
		}

		if ref.Peername != "" && ref.PeerID != "" {
			if ref.Peername == p.Peername && ref.PeerID != p.ID {
				return fmt.Errorf("Peername and PeerID combination not valid: Peername = %s, PeerID = %s, but was given PeerID = %s", p.Peername, p.ID, ref.PeerID)
			}
			if ref.PeerID == p.ID && ref.Peername != p.Peername {
				return fmt.Errorf("Peername and PeerID combination not valid: PeerID = %s, Peername = %s, but was given Peername = %s", p.ID, p.Peername, ref.Peername)
			}
			if ref.Peername == p.Peername && ref.PeerID == p.ID {
				return nil
			}
		}

		if ref.Peername != "" {
			if ref.Peername != p.Peername {
				return nil
			}
		}

		if ref.PeerID != "" {
			if ref.PeerID != p.ID {
				return nil
			}
		}

		ref.Peername = p.Peername
		ref.PeerID = p.ID
		return nil
	}
	if ref.PeerID != "" {
		pid, err := profile.NewB58PeerID(ref.PeerID)
		if err != nil {
			return fmt.Errorf("error converting PeerID to base58 hash: %s", err)
		}

		peer, err := r.Profiles().GetPeer(pid)
		if err != nil {
			return fmt.Errorf("error fetching peers from store: %s", err)
		}

		if ref.Peername == "" {
			ref.Peername = peer.Peername
			return nil
		}
		if ref.Peername != peer.Peername {
			return fmt.Errorf("Peername and PeerID combination not valid: PeerID = %s, Peername = %s, but was given Peername = %s", peer.ID, peer.Peername, ref.Peername)
		}
	}

	if ref.Peername != "" {
		id, err := r.Profiles().GetID(ref.Peername)
		if err != nil {
			return fmt.Errorf("error fetching peer from store: %s", err)
		}
		if ref.PeerID == "" {
			ref.PeerID = id.String()
			return nil
		}
		if ref.PeerID != id.String() {
			return fmt.Errorf("Peername and PeerID combination not valid: Peername = %s, PeerID = %s, but was given PeerID = %s", ref.Peername, id.String(), ref.PeerID)
		}
	}
	return nil
}

// CompareDatasetRef compares two Dataset References, returning an error
// describing any difference between the two references
func CompareDatasetRef(a, b DatasetRef) error {
	if a.PeerID != b.PeerID {
		return fmt.Errorf("peerID mismatch. %s != %s", a.PeerID, b.PeerID)
	}
	if a.Peername != b.Peername {
		return fmt.Errorf("peername mismatch. %s != %s", a.Peername, b.Peername)
	}
	if a.Name != b.Name {
		return fmt.Errorf("name mismatch. %s != %s", a.Name, b.Name)
	}
	if a.Path != b.Path {
		return fmt.Errorf("path mismatch. %s != %s", a.Path, b.Path)
	}
	return nil
}
