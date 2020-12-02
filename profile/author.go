package profile

import (
	"fmt"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multihash"
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

func (a author) AuthorPubKeyID() crypto.PubKey {
	return a.pubKey
}

func (a author) AuthorPubKey() crypto.PubKey {
	return a.pubKey
}

func (a author) Username() string {
	return a.username
}

// KeyIDFromPriv is a wrapper for calling KeyIDFromPub on a private key
func KeyIDFromPriv(pk crypto.PrivKey) (string, error) {
	return KeyIDFromPub(pk.GetPublic())
}

// KeyIDFromPub returns the base58btc-encoded multihash string of a public key
// hash. hashes are 32-byte sha2-256 sums of public key bytes
// KeyID is a unique identifier for a keypair
func KeyIDFromPub(pubKey crypto.PubKey) (string, error) {
	if pubKey == nil {
		return "", fmt.Errorf("identity: public key is required")
	}

	pubkeybytes, err := pubKey.Bytes()
	if err != nil {
		return "", fmt.Errorf("getting pubkey bytes: %s", err.Error())
	}

	mh, err := multihash.Sum(pubkeybytes, multihash.SHA2_256, 32)
	if err != nil {
		return "", fmt.Errorf("summing pubkey: %s", err.Error())
	}

	return mh.B58String(), nil
}
