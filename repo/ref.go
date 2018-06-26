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

// ErrRefSelectionNotSupported is the expected error for when RefSelector interface is *not* implemented
var ErrRefSelectionNotSupported = fmt.Errorf("selection not supported")

// RefSelector is an interface for supporting reference selection
// a reference selection is a slice of references intended for using
// in dataset operations
type RefSelector interface {
	SetSelectedRefs([]DatasetRef) error
	SelectedRefs() ([]DatasetRef, error)
}

// ProfileRef encapsulates a reference to a peer profile
// It's main job is to connect peernames / profile ID's to profiles
type ProfileRef struct {
	Peername  string     `json:"peername,omitempty"`
	ProfileID profile.ID `json:"profileID,omitempty"`
	// Profile data
	Profile *profile.Profile
}

// String implements the Stringer interface for PeerRef
func (r ProfileRef) String() (s string) {
	s = r.Peername
	if r.ProfileID.String() != "" {
		s += "@" + r.ProfileID.String()
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
	// ProfileID of dataset owner
	ProfileID profile.ID `json:"profileID,omitempty"`
	// Unique name reference for this dataset
	Name string `json:"name,omitempty"`
	// Content-addressed path for this dataset
	Path string `json:"path,omitempty"`
	// Dataset is a pointer to the dataset being referenced
	Dataset *dataset.DatasetPod `json:"dataset,omitempty"`
}

// DecodeDataset returns a dataset.Dataset from the stored CodingDataset field
func (r DatasetRef) DecodeDataset() (*dataset.Dataset, error) {
	if r.Dataset == nil {
		return nil, nil
	}
	ds := &dataset.Dataset{}
	return ds, ds.Decode(r.Dataset)
}

// String implements the Stringer interface for DatasetRef
func (r DatasetRef) String() (s string) {
	s = r.AliasString()
	if r.ProfileID.String() != "" || r.Path != "" {
		s += "@"
	}
	if r.ProfileID.String() != "" {
		s += r.ProfileID.String()
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
	return (r.Path != "" && b.Path != "" && r.Path == b.Path) || (r.ProfileID == b.ProfileID || r.Peername == b.Peername) && r.Name == b.Name
}

// Equal returns true only if Peername Name and Path are equal
func (r DatasetRef) Equal(b DatasetRef) bool {
	return r.Peername == b.Peername && r.ProfileID == b.ProfileID && r.Name == b.Name && r.Path == b.Path
}

// IsPeerRef returns true if only Peername is set
func (r DatasetRef) IsPeerRef() bool {
	return (r.Peername != "" || r.ProfileID != "") && r.Name == "" && r.Path == "" && r.Dataset == nil
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
		dsr.ProfileID, dsr.Path, err = parseIdentifiers(ref[atIndex+1:])

	} else {

		var hasFirst, hasSecond, hasPid bool
		var first, second string
		toks := strings.Split(ref, "/")

		for i, tok := range toks {
			if isBase58Multihash(tok) {
				// first hash we encounter is a peerID
				if !hasPid {
					dsr.ProfileID, _ = profile.IDB58Decode(tok)
					hasPid = true
					continue
				}

				if !isBase58Multihash(toks[i-1]) {
					dsr.Path = fmt.Sprintf("/%s/%s", toks[i-1], strings.Join(toks[i:], "/"))
				} else {
					dsr.Path = fmt.Sprintf("/ipfs/%s", strings.Join(toks[i:], "/"))
				}
				break
			}

			if !hasFirst {
				first = tok
				hasFirst = true
				continue
			}

			if !hasSecond {
				second = tok
				hasSecond = true
				continue
			}

			dsr.Path = strings.Join(toks[i:], "/")
			break
		}

		if hasFirst && !hasSecond {
			dsr.Name = first
		} else if hasFirst && hasSecond {
			dsr.Peername = first
			dsr.Name = second
		}
	}

	if dsr.ProfileID == "" && dsr.Peername == "" && dsr.Name == "" && dsr.Path == "" {
		err = fmt.Errorf("malformed DatasetRef string: %s", ref)
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
		err = fmt.Errorf("malformed DatasetRef identifier: %s", ids)
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

// CanonicalizeDatasetRef uses a repo to turn any local aliases into known
// canonical peername for a dataset and populates a missing path
// if the repo has path information for a peername/name combo
// if we provide any other shortcuts for names other than "me"
// in the future, it should be handled here.
func CanonicalizeDatasetRef(r Repo, ref *DatasetRef) error {
	// when operating over RPC there's a good chance we won't have a repo, in that
	// case we're going to have to rely on the other end of the wire to do canonicalization
	// TODO - think carefully about placement of reference parsing, possibly moving
	// this into lib functions.
	if r == nil {
		return nil
	}

	if err := CanonicalizeProfile(r, ref); err != nil {
		return err
	}

	if ref.Path != "" && ref.ProfileID != "" && ref.Name != "" && ref.Peername != "" {
		return nil
	}

	got, err := r.GetRef(*ref)
	if err == nil {
		if ref.Path == "" {
			ref.Path = got.Path
		}
		if ref.ProfileID == "" {
			ref.ProfileID = got.ProfileID
		}
		if ref.Name == "" {
			ref.Name = got.Name
		}
		if ref.Peername == "" {
			ref.Peername = got.Peername
		}
		if ref.Path != got.Path || ref.ProfileID != got.ProfileID || ref.Name != got.Name || ref.Peername != got.Peername {
			return fmt.Errorf("Given datasetRef %s does not match datasetRef on file: %s", ref.String(), got.String())
		}
	}

	return nil
}

// CanonicalizeProfile populates dataset DatasetRef ProfileID and Peername properties,
// changing aliases to known names, and adding ProfileID from a peerstore
func CanonicalizeProfile(r Repo, ref *DatasetRef) error {
	if ref.Peername == "" && ref.ProfileID == "" {
		return nil
	}

	p, err := r.Profile()
	if err != nil {
		return err
	}

	if ref.Peername == "me" || ref.Peername == p.Peername || ref.ProfileID == p.ID {
		if ref.Peername == "me" {
			ref.ProfileID = p.ID
			ref.Peername = p.Peername
		}

		if ref.Peername != "" && ref.ProfileID != "" {
			if ref.Peername == p.Peername && ref.ProfileID != p.ID {
				return fmt.Errorf("Peername and ProfileID combination not valid: Peername = %s, ProfileID = %s, but was given ProfileID = %s", p.Peername, p.ID, ref.ProfileID)
			}
			if ref.ProfileID == p.ID && ref.Peername != p.Peername {
				return fmt.Errorf("Peername and ProfileID combination not valid: ProfileID = %s, Peername = %s, but was given Peername = %s", p.ID, p.Peername, ref.Peername)
			}
			if ref.Peername == p.Peername && ref.ProfileID == p.ID {
				return nil
			}
		}

		if ref.Peername != "" {
			if ref.Peername != p.Peername {
				return nil
			}
		}

		if ref.ProfileID != "" {
			if ref.ProfileID != p.ID {
				return nil
			}
		}

		ref.Peername = p.Peername
		ref.ProfileID = p.ID
		return nil
	}
	if ref.ProfileID != "" {
		// pid, err := profile.NewB58ID(ref.ProfileID)
		// if err != nil {
		// 	return fmt.Errorf("error converting ProfileID to base58 hash: %s", err)
		// }

		profile, err := r.Profiles().GetProfile(ref.ProfileID)
		if err != nil {
			return fmt.Errorf("error fetching peers from store: %s", err)
		}

		if ref.Peername == "" {
			ref.Peername = profile.Peername
			return nil
		}
		if ref.Peername != profile.Peername {
			return fmt.Errorf("Peername and ProfileID combination not valid: ProfileID = %s, Peername = %s, but was given Peername = %s", profile.ID, profile.Peername, ref.Peername)
		}
	}

	if ref.Peername != "" {
		id, err := r.Profiles().PeernameID(ref.Peername)
		if err != nil {
			return fmt.Errorf("error fetching peer from store: %s", err)
		}
		if ref.ProfileID == "" {
			ref.ProfileID = id
			return nil
		}
		if ref.ProfileID != id {
			return fmt.Errorf("Peername and ProfileID combination not valid: Peername = %s, ProfileID = %s, but was given ProfileID = %s", ref.Peername, id.String(), ref.ProfileID)
		}
	}
	return nil
}

// CompareDatasetRef compares two Dataset References, returning an error
// describing any difference between the two references
func CompareDatasetRef(a, b DatasetRef) error {
	if a.ProfileID != b.ProfileID {
		return fmt.Errorf("peerID mismatch. %s != %s", a.ProfileID, b.ProfileID)
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
