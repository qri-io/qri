// Package repo represents a repository of qri information
// Analogous to a git repository, repo expects a rigid structure
// filled with rich types specific to qri.
// Lots of things in here take inspiration from the ipfs datastore interface:
// github.com/ipfs/go-datastore
package repo

import (
	"fmt"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry/regclient"
)

var (
	// ErrNotFound is the err implementers should return when stuff isn't found
	ErrNotFound = fmt.Errorf("repo: not found")
	// ErrPeerIDRequired is for when a peerID is missing-but-expected
	ErrPeerIDRequired = fmt.Errorf("repo: peerID is required")
	// ErrPeernameRequired is for when a peername is missing-but-expected
	ErrPeernameRequired = fmt.Errorf("repo: peername is required")
	// ErrNameRequired is for when a name is missing-but-expected
	ErrNameRequired = fmt.Errorf("repo: name is required")
	// ErrPathRequired is for when a path is missing-but-expected
	ErrPathRequired = fmt.Errorf("repo: path is required")
	// ErrNameTaken is for when a name name is already taken
	ErrNameTaken = fmt.Errorf("repo: name already in use")
	// ErrRepoEmpty is for when the repo has no datasets
	ErrRepoEmpty = fmt.Errorf("repo: this repo contains no datasets")
	// ErrNotPinner is for when the repo doesn't have the concept of pinning as a feature
	ErrNotPinner = fmt.Errorf("repo: backing store doesn't support pinning")
	// ErrNoRegistry indicates no regsitry is currently configured
	ErrNoRegistry = fmt.Errorf("no configured registry")
	// ErrEmptyRef indicates that the given reference is empty
	ErrEmptyRef = fmt.Errorf("repo: empty dataset reference")
)

// Repo is the interface for working with a qri repository qri repos are stored
// graph of resources:datasets, known peers, analytics data, change requests, etc.
// Repos are connected to a single peer profile.
// Repos must wrap an underlying cafs.Filestore, which
// is intended to act as the canonical store of state across all peers
// that this repo may interact with.
type Repo interface {
	// All repositories wraps a content-addressed filestore as the cannonical
	// record of this repository's data. Store gives direct access to the
	// cafs.Filestore instance any given repo is using.
	Store() cafs.Filestore

	// All Repos must keep a Refstore, defining a store of known datasets
	Refstore
	// EventLog keeps a log of Profile activity for this repo
	EventLog

	// A repository must maintain profile information about the owner of this dataset.
	// The value returned by Profile() should represent the peer.
	Profile() (*profile.Profile, error)
	// SetProfile sets this repo's current profile. Profiles must contain a private key
	SetProfile(*profile.Profile) error
	// PrivateKey hands over this repo's private key
	// TODO - this is needed to create action structs, any way we can make this
	// privately-negotiated or created at init?
	PrivateKey() crypto.PrivKey
	// A repository must maintain profile information about encountered peers.
	// Decsisions regarding retentaion of peers is left to the the implementation
	// TODO - should rename this to "profiles" to separate from the networking
	// concept of a peer
	Profiles() profile.Store
	// Registry returns a client for interacting with a federated registry
	// if one exists, otherwise nil
	Registry() *regclient.Client
}

// SearchParams encapsulates parameters provided to Searchable.Search
type SearchParams struct {
	Q             string
	Limit, Offset int
}

// Searchable is an opt-in interface for supporting repository search
type Searchable interface {
	Search(p SearchParams) ([]DatasetRef, error)
}
