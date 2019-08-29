/*
Package registry defines primitives for keeping centralized repositories
of qri types (peers, datasets, etc). It uses classical client/server patterns,
arranging types into cannonical stores.

At first glance, this seems to run against the grain of "decentralize or die"
principles espoused by those of us interested in reducing points of failure in
a network. Registries offer a way to operate as a federated model, with peers
opting-in to a set of norms set forth by a registry.

It is a long term goal at qri that it be *possible* to fully decentralize all
aspects, of qri this isn't practical short-term, and isn't always a desired
property.

As an example, associating human-readable usernames with crypto keypairs is an
order of magnitude easier if you just put the damn thing in a list. So that's
what this registry does.

This base package provides common primitives that other packages can import to
work with a registry, and subpackages for turning these primitives into usable
tools like servers & (eventually) command-line clients
*/
package registry

import "fmt"

// Registry a collection of interfaces that together form a registry service
type Registry struct {
	Profiles    Profiles
	Reputations Reputations
	Search      Searchable
	Indexer     Indexer
}

var (
	// ErrUsernameTaken is for when a peername is already taken
	ErrUsernameTaken = fmt.Errorf("username is taken")
	// ErrNoRegistry represents the lack of a configured registry
	ErrNoRegistry = fmt.Errorf("no registry is configured")
	// ErrNotFound represents a missing record
	ErrNotFound = fmt.Errorf("not found")
)
