package key

import (
	"encoding/base64"
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

// EncodePubKeyB64 serializes a public key to a base64-encoded string
func EncodePubKeyB64(pub crypto.PubKey) (string, error) {
	if pub == nil {
		return "", fmt.Errorf("cannot encode nil public key")
	}
	pubBytes, err := pub.Bytes()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pubBytes), nil
}

// DecodeB64PubKey deserializes a base-64 encoded key string into a public key
func DecodeB64PubKey(keystr string) (pk crypto.PubKey, err error) {
	d, err := base64.StdEncoding.DecodeString(keystr)
	if err != nil {
		return nil, fmt.Errorf("decoding base64-encoded public key: %w", err)
	}
	pubKey, err := crypto.UnmarshalPublicKey(d)
	if err != nil {
		return nil, fmt.Errorf("public key %q is invalid: %w", keystr, err)
	}
	return pubKey, nil
}

// EncodePrivKeyB64 serializes a private key to a base64-encoded string
func EncodePrivKeyB64(pk crypto.PrivKey) (string, error) {
	if pk == nil {
		return "", fmt.Errorf("cannot encode nil private key")
	}
	pdata, err := pk.Bytes()
	return base64.StdEncoding.EncodeToString(pdata), err
}

// DecodeB64PrivKey deserializes a base-64 encoded key string into a private
// key
func DecodeB64PrivKey(keystr string) (pk crypto.PrivKey, err error) {
	data, err := base64.StdEncoding.DecodeString(keystr)
	if err != nil {
		return nil, fmt.Errorf("decoding base64-encoded private key: %w", err)
	}

	pk, err = crypto.UnmarshalPrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	return pk, nil
}
