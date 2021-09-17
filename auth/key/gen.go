package key

import (
	"crypto/rand"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
)

// CryptoGenerator is an interface for generating cryptographic info like
// private keys and peerIDs
// TODO(b5): I've moved this here because the key package should be the source
// of all cryptographic data, but this needs work. I'd like to see it reduced
// to just a `GeneratePrivateKey` function
type CryptoGenerator interface {
	// GeneratePrivateKeyAndPeerID returns a base64 encoded private key, and a
	// peerID
	GeneratePrivateKeyAndPeerID() (string, string)
}

// cryptoGenerator is a source of cryptographic info
type cryptoGenerator struct{}

var _ CryptoGenerator = (*cryptoGenerator)(nil)

// NewCryptoGenerator returns a source of p2p cryptographic info that
// performs expensive computations like repeated primality testing
func NewCryptoGenerator() CryptoGenerator {
	return &cryptoGenerator{}
}

// GeneratePrivateKeyAndPeerID returns a private key and peerID
func (g cryptoGenerator) GeneratePrivateKeyAndPeerID() (privKey, peerID string) {
	r := rand.Reader
	// Generate a key pair for this host. This is a relatively expensive operation.
	if priv, pub, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r); err == nil {
		privKey, err = EncodePrivKeyB64(priv)
		if err != nil {
			panic(err)
		}
		// Obtain peerID from public key
		if pid, err := peer.IDFromPublicKey(pub); err == nil {
			peerID = pid.Pretty()
		}
	}
	return
}
