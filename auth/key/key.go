package key

import (
	"fmt"

	logger "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multihash"
)

var log = logger.Logger("key")

// ID is a key identifier
// note that while this implementation is an alias for a peer.ID, this is not
// strictly a peerID.
type ID = peer.ID

// DecodeID parses an ID string
func DecodeID(s string) (ID, error) {
	pid, err := peer.Decode(s)
	if err != nil {
		return "", err
	}
	return ID(pid), nil
}

// IDFromPrivKey is a wrapper for calling IDFromPubKey on a private key
func IDFromPrivKey(pk crypto.PrivKey) (string, error) {
	return IDFromPubKey(pk.GetPublic())
}

// IDFromPubKey returns the base58btc-encoded multihash string of a public key
// hash. hashes are 32-byte sha2-256 sums of public key bytes
// KeyID is a unique identifier for a keypair
func IDFromPubKey(pubKey crypto.PubKey) (string, error) {
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
