package repo

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mr-tron/base58/base58"
	"github.com/multiformats/go-multihash"
	"github.com/qri-io/dataset"
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
	// Unique name reference for this dataset
	Name string `json:"name,omitempty"`
	// Content-addressed path for this dataset
	Path string `json:"path,omitempty"`
	// Dataset is a pointer to the dataset being referenced
	Dataset *dataset.Dataset `json:"dataset,omitempty"`
}

// String implements the Stringer interface for DatasetRef
func (r DatasetRef) String() (s string) {
	s = r.Peername
	if r.Name != "" {
		s += "/" + r.Name
	}
	if r.Path != "" {
		s += "@" + strings.TrimLeft(r.Path, "/")
	}
	return s
}

// Match checks returns true if Peername and Name are equal,
// and/or path is equal
func (r DatasetRef) Match(b DatasetRef) bool {
	// fmt.Printf("\nr.Peername: %s b.Peername: %s\n", r.Peername, b.Peername)
	// fmt.Printf("\nr.Name: %s b.Name: %s\n", r.Name, b.Name)
	return r.Peername == b.Peername && r.Name == b.Name || r.Path == b.Path
}

// Equal returns true only if Peername Name and Path are equal
func (r DatasetRef) Equal(b DatasetRef) bool {
	return r.Peername == b.Peername && r.Name == b.Name && r.Path == b.Path
}

// IsPeerRef returns true if only Peername is set
func (r DatasetRef) IsPeerRef() bool {
	return r.Peername != "" && r.Name == "" && r.Path == "" && r.Dataset == nil
}

// IsEmpty returns true if none of it's fields are set
func (r DatasetRef) IsEmpty() bool {
	return r.Equal(DatasetRef{})
}

var (
	// fullDatasetPathRegex looks for dataset references in the forms
	// peername/dataset_name@/ipfs/hash
	fullDatasetPathRegex = regexp.MustCompile(`(\w+)/(\w+)@(/\w+/)(\w+)\b`)
	// hashShorthandPathRegex looks for dataset references in the forms:
	// peername/dataset_name@hash
	// peername/dataset_name@/hash
	hashShorthandPathRegex = regexp.MustCompile(`(\w+)/(\w+)@/?(\w+)\b`)

	// peernameShorthandPathRegex looks for dataset references in the form:
	// peername/dataset_name
	peernameShorthandPathRegex = regexp.MustCompile(`(\w+)/(\w+)$`)

	// fullnameRegex looks for dataset references in the form:
	// peername/dataset_name
	fullnameRegex = regexp.MustCompile(`(\w+)/(\w+)$`)

	// simpleRegex looks for the first word in a string
	simpleRegex = regexp.MustCompile(`(\w+)$`)

	// pathRegex looks for dataset references in the form:
	// network/hash/etc
	pathRegex = regexp.MustCompile(`(\w+)/(\w+)`)
)

// ParseDatasetRef decodes a dataset reference from a string value
// It’s possible to refer to a dataset in a number of ways.
// The full definition of a dataset reference is as follows:
//     dataset_reference = peer_name/dataset_name@network/hash
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
// through defaults the following should all parse:
//     peer_name/dataset_name
//     dataset_name@hash
//     /network/hash
//     peername
//     hash
//
// see tests for more exmples
//
// dataset names & hashes are disambiguated by checking if the input
// parses to a valid multihash after base58 decoding
//
// TODO - add validation that prevents peernames from being
// valid base58 multihashes and makes sure hashes are actually valid base58 multihashes
// TODO - figure out how IPFS CID's play into this
func ParseDatasetRef(ref string) (DatasetRef, error) {
	if ref == "" {
		return DatasetRef{}, fmt.Errorf("cannot parse empty string as dataset reference")
	}
	var (
		nameRefString string
		pathRefString string
		path          string
		peername      string
		datasetname   string
	)
	// if there is an @ symbol, we are dealing with a DatasetRef
	// with a specific path
	atIndex := strings.Index(ref, "@")
	if atIndex != -1 {
		nameRefString = strings.Trim(ref[:atIndex], "/")
		pathRefString = strings.Trim(ref[atIndex+1:], "/")
	} else {
		nameRefString = strings.Trim(ref, "/")
		if nameRefString == "" {
			return DatasetRef{}, fmt.Errorf("malformed DatasetRef string: %s", ref)
		}
	}
	if pathRefString != "" {
		network := ""
		hash := ""
		if pathRegex.MatchString(pathRefString) {
			matches := pathRegex.FindStringSubmatch(pathRefString)
			network = matches[1]
			hash = matches[2]
			if !isBase58Multihash(hash) {
				if isBase58Multihash(network) {
					hash = network
					network = "ipfs"
				} else {
					return DatasetRef{}, fmt.Errorf("'%s' is not a base58 multihash", pathRefString)
				}
			}
		} else if simpleRegex.MatchString(pathRefString) {
			matches := simpleRegex.FindStringSubmatch(pathRefString)
			hash = matches[1]
			network = "ipfs"
			if !isBase58Multihash(hash) {
				return DatasetRef{}, fmt.Errorf("'%s' is not a base58 multihash", pathRefString)
			}
		} else {
			return DatasetRef{}, fmt.Errorf("malformed DatasetRef string: %s", ref)
		}
		path = "/" + network + "/" + hash
	}
	if nameRefString != "" {
		if fullnameRegex.MatchString(nameRefString) {
			matches := fullnameRegex.FindStringSubmatch(nameRefString)
			peername = matches[1]
			datasetname = matches[2]
		} else if simpleRegex.MatchString(nameRefString) {
			matches := simpleRegex.FindStringSubmatch(nameRefString)
			peername = matches[1]
		} else {
			return DatasetRef{}, fmt.Errorf("malformed DatasetRef string: %s", ref)
		}
	}

	return DatasetRef{
		Peername: peername,
		Name:     datasetname,
		Path:     path,
	}, nil

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

	if err := CanonicalizePeername(r, &ref.Peername); err != nil {
		return err
	}

	// Proactively attempt to find dataset path
	if ref.Path == "" {
		if got, err := r.GetRef(*ref); err == nil {
			*ref = got
			return nil
		}
	}
	return nil
}

// CanonicalizePeername uses a repo to replace aliases with
// canonical peernames. basically, this thing replaces "me" with the proper peername.
func CanonicalizePeername(r Repo, peername *string) error {
	if *peername == "me" {
		p, err := r.Profile()
		if err != nil {
			return err
		}
		*peername = p.Peername
	}
	return nil
}

// CompareDatasetRef compares two Dataset References, returning an error
// describing any difference between the two references
func CompareDatasetRef(a, b DatasetRef) error {
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
