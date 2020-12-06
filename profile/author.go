package profile

import (
	"github.com/libp2p/go-libp2p-core/crypto"
)

// Author uses keypair cryptography to distinguish between different log sources
// (authors)
//
// Deprecated - don't rely on this interface, refactor to use profile.Profiles
// and public keys instead
type Author interface {
	AuthorID() string
	AuthorPubKey() crypto.PubKey
	Username() string
}

type author struct {
	id       string
	pubKey   crypto.PubKey
	username string
}

// NewAuthor creates an Author interface implementation, allowing outside
// packages needing to satisfy the Author interface
//
// Deprecated - use profile.Profile instead
func NewAuthor(id string, pubKey crypto.PubKey, username string) Author {
	return author{
		id:       id,
		pubKey:   pubKey,
		username: username,
	}
}

func (a author) AuthorID() string {
	return a.id
}

func (a author) AuthorPubKey() crypto.PubKey {
	return a.pubKey
}

func (a author) Username() string {
	return a.username
}
