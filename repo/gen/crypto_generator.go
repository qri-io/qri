// Package gen contains routines that perform expensive cryptographic
// operations. These should only be used when absolutely needed by
// top-level commands, and not in, for example, in test code.
package gen

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	ipfs "github.com/qri-io/qfs/cafs/ipfs"
	"github.com/qri-io/doggos"
)

// CryptoGenerator is an interface for generating cryptographic info like private keys and peerIDs
type CryptoGenerator interface {
	// GeneratePrivateKeyAndPeerID returns a base64 encoded private key, and a peerID
	GeneratePrivateKeyAndPeerID() (string, string)
	// GenerateNickname uses a peerID to return a human-friendly nickname
	GenerateNickname(peerID string) string
	// GenerateEmptyIpfsRepo creates an empty IPFS repo at a given path
	GenerateEmptyIpfsRepo(repoPath, cfgPath string) error
}

// CryptoSource is a source of cryptographic info
type CryptoSource struct {
}

// NewCryptoSource returns a source of p2p cryptographic info that
// performs expensive computations like repeated primality testing
func NewCryptoSource() *CryptoSource {
	return &CryptoSource{}
}

// GeneratePrivateKeyAndPeerID returns a private key and peerID
func (g *CryptoSource) GeneratePrivateKeyAndPeerID() (privKey, peerID string) {
	r := rand.Reader
	// Generate a key pair for this host. This is a relatively expensive operation.
	if priv, pub, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r); err == nil {
		if pdata, err := priv.Bytes(); err == nil {
			privKey = base64.StdEncoding.EncodeToString(pdata)
		}
		// Obtain peerID from public key
		if pid, err := peer.IDFromPublicKey(pub); err == nil {
			peerID = pid.Pretty()
		}
	}
	return
}

// GenerateNickname returns a nickname using a peerID as a seed
func (g *CryptoSource) GenerateNickname(peerID string) string {
	return doggos.DoggoNick(peerID)
}

// GenerateEmptyIpfsRepo creates an empty IPFS repo in a secure manner at the given path
func (g *CryptoSource) GenerateEmptyIpfsRepo(repoPath, configPath string) error {
	return ipfs.InitRepo(repoPath, configPath)
}
