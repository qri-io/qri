package key

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
)

// ID identifies a key, usually it's determined by hash-of-publickey
type ID = peer.ID

// Public is the public half of a cryptographic key, plus an identifier
type Public struct {
	ID  ID
	Key crypto.PubKey
}

// type Private struct {
// 	ID ID
// 	Key crypto.PrivKey
// }
