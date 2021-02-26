package key

import (
	"fmt"

	logger "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multihash"
)

var log = logger.Logger("key")

// ID is the key identifier
type ID string

// IDFromPriv is a wrapper for calling KeyIDFromPub on a private key
func IDFromPriv(pk crypto.PrivKey) (string, error) {
	return IDFromPub(pk.GetPublic())
}

// IDFromPub returns the base58btc-encoded multihash string of a public key
// hash. hashes are 32-byte sha2-256 sums of public key bytes
// KeyID is a unique identifier for a keypair
func IDFromPub(pubKey crypto.PubKey) (string, error) {
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
