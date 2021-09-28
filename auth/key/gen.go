package key

import (
	"crypto/rand"

	"github.com/libp2p/go-libp2p-core/crypto"
	crypto_pb "github.com/libp2p/go-libp2p-core/crypto/pb"
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

// cryptoGenerator is a source of cryptographic info for RSA keys
type cryptoGenerator struct {
	algo crypto_pb.KeyType
	bits int
}

var _ CryptoGenerator = (*cryptoGenerator)(nil)

// NewCryptoGenerator returns the default source of p2p cryptographic info that
// performs expensive computations like repeated primality testing
func NewCryptoGenerator() CryptoGenerator {
	return &cryptoGenerator{
		algo: crypto.Ed25519,
		bits: 0,
	}
}

// NewRSACryptoGenerator returns a source of RSA based p2p cryptographic info that
// performs expensive computations like repeated primality testing
func NewRSACryptoGenerator() CryptoGenerator {
	return &cryptoGenerator{
		algo: crypto.RSA,
		bits: 2048,
	}
}

// GeneratePrivateKeyAndPeerID returns a private key and peerID
func (g cryptoGenerator) GeneratePrivateKeyAndPeerID() (privKey, peerID string) {
	r := rand.Reader
	// Generate a key pair for this host. This is a relatively expensive operation.
	if priv, pub, err := crypto.GenerateKeyPairWithReader(int(g.algo), g.bits, r); err == nil {
		privKey, err = EncodePrivKeyB64(priv)
		if err != nil {
			panic(err)
		}
		// Obtain profile.ID from public key
		if pid, err := IDFromPubKey(pub); err == nil {
			peerID = pid
		}
	}
	return
}
