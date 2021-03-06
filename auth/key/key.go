package key

import (
	"fmt"

	logger "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
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

// IDFromPubKey returns an ID string is a that is unique identifier for a
// keypair. For RSA keys this is the base58btc-encoded multihash string of
// the public key. hashes are 32-byte sha2-256 sums of public key bytes
func IDFromPubKey(pubKey crypto.PubKey) (string, error) {
	if pubKey == nil {
		return "", fmt.Errorf("public key is required")
	}

	id, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return "", err
	}
	return id.Pretty(), err
}
