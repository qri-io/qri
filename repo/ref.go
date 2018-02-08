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
		s += "@" + r.Path
	}
	return s
}

// Match checks returns true if Peername and Name are equal,
// and/or path is equal
func (r DatasetRef) Match(b DatasetRef) bool {
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

var (
	// fullDatasetPathRegex looks for dataset references in the forms:
	// peername/dataset_name@/ipfs/hash
	// peername/dataset_name@hash
	fullDatasetPathRegex = regexp.MustCompile(`(\w+)/(\w+)@(/\w+/)?(\w+)\b`)
	// peernameShorthandPathRegex looks for dataset references in the form:
	// peername/dataset_name
	peernameShorthandPathRegex = regexp.MustCompile(`(\w+)/(\w+)$`)
)

// ParseDatasetRef decodes a dataset reference from a string value
// It’s possible to refer to a dataset in a number of ways.
// The full definition of a dataset reference is as follows:
//     dataset_reference = peer_name/dataset_name@/network/hash
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
// valid base58 multihashes.
// TODO - figure out how IPFS CID's play into this
func ParseDatasetRef(ref string) (DatasetRef, error) {
	if ref == "" {
		return DatasetRef{}, fmt.Errorf("cannot parse empty string as dataset reference")
	} else if strings.HasPrefix(ref, "/ipfs/") {
		return DatasetRef{
			Path: ref,
		}, nil
	} else if fullDatasetPathRegex.MatchString(ref) {
		matches := fullDatasetPathRegex.FindAllStringSubmatch(ref, 1)
		if matches[0][3] == "" {
			matches[0][3] = "/ipfs/"
		}
		return DatasetRef{
			Peername: matches[0][1],
			Name:     matches[0][2],
			Path:     matches[0][3] + matches[0][4],
		}, nil
	} else if peernameShorthandPathRegex.MatchString(ref) {
		matches := peernameShorthandPathRegex.FindAllStringSubmatch(ref, 1)
		return DatasetRef{
			Peername: matches[0][1],
			Name:     matches[0][2],
		}, nil
	}

	if data, err := base58.Decode(stripProtocol(stripProtocol(ref))); err == nil {
		if _, err := multihash.Decode(data); err == nil {
			return DatasetRef{
				Path: "/ipfs/" + stripProtocol(ref),
			}, nil
		}
	}

	return DatasetRef{
		Peername: ref,
	}, nil
}

// IsLocalRef checks to see if a given reference needs to be
// resolved against the network
func IsLocalRef(r Repo, ref DatasetRef) (bool, error) {
	if ref.Peername == "" || ref.Peername == "me" {
		return true, nil
	}

	p, err := r.Profile()
	if err != nil {
		return false, err
	}

	// TODO - check to see if repo has local / cached copy
	// of the reference in question

	return ref.Peername == p.Peername, nil
}

// TODO - this could be more robust?
func stripProtocol(ref string) string {
	if strings.HasPrefix(ref, "/ipfs/") {
		return ref[len("/ipfs/"):]
	}
	return ref
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
