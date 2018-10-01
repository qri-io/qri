package gen

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/qri-io/doggos"
)

// CryptoGenerator is an interface for generating cryptographic info like private keys and peerIDs
type CryptoGenerator interface {
	GeneratePrivateKeyAndPeerID() (string, string)
	GenerateNickname(peerID string) string
}

// RealCryptoSource is a source of cryptographic info
type RealCryptoSource struct {
}

// NewCryptoSource returns a real source of p2p cryptographic info
func NewCryptoSource() *RealCryptoSource {
	return &RealCryptoSource{}
}

// GeneratePrivateKeyAndPeerID returns a private key and peerID
func (g *RealCryptoSource) GeneratePrivateKeyAndPeerID() (string, string) {
	var privKey, peerID string

	r := rand.Reader
	// Generate a key pair for this host. This is a relatively expensive operation.
	if priv, pub, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r); err == nil {
		if pdata, err := priv.Bytes(); err == nil {
			privKey = base64.StdEncoding.EncodeToString(pdata)
		}

		// Obtain Peer ID from public key
		if pid, err := peer.IDFromPublicKey(pub); err == nil {
			peerID = pid.Pretty()
		}
	}

	return privKey, peerID
}

// GenerateNickname returns a nickname using a peerID as a seed
func (g *RealCryptoSource) GenerateNickname(peerID string) string {
	return doggos.DoggoNick(peerID)
}

// TODO: Add a method that wraps the behavior of cafs/ipfs/init, which also performs
// a bunch of cryptographic generation.
