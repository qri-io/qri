package key

import (
	"encoding/base64"
	"fmt"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-peerstore/pstoremem"
	"github.com/qri-io/qri/config"
)

// Store is an abstraction over a libp2p.KeyBook
// In the future we may expand this interface to store symmetric encryption keys
type Store interface {
	peerstore.KeyBook

	Owner() (peer.ID, crypto.PrivKey)
}

// NewStore constructs a keys.Store backed by memory
func NewStore(cfg *config.Config) (Store, error) {
	if cfg.Profile == nil {
		return nil, fmt.Errorf("profile is required")
	}
	if cfg.Profile.PrivKey == "" {
		return nil, fmt.Errorf("profile private key is required")
	}

	data, err := base64.StdEncoding.DecodeString(cfg.Profile.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("decoding private key: %w", err)
	}

	pk, err := crypto.UnmarshalPrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	id, err := peer.IDB58Decode(cfg.Profile.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid owner ID: %w", err)
	}

	return NewMemStore(id, pk)
	// switch cfg.Repo.Type {
	// case "fs":
	// case "mem":
	// 	return NewMemStore(pro)
	// default:
	// 	// return nil, fmt.Errorf("unknown repo type: %s", cfg.Repo.Type)
	// }
}

type memStore struct {
	peerstore.KeyBook
	ownerID  peer.ID
	ownerKey crypto.PrivKey
}

// NewMemStore constructs an in-memory keys.Store
func NewMemStore(ownerID peer.ID, ownerKey crypto.PrivKey) (Store, error) {
	return NewMemStoreKeybook(ownerID, ownerKey, pstoremem.NewKeyBook())
}

// NewMemStoreKeybook creates an in-memory keys.Store with a custom keybook
func NewMemStoreKeybook(ownerID peer.ID, ownerKey crypto.PrivKey, kb peerstore.KeyBook) (Store, error) {
	return &memStore{
		ownerID:  ownerID,
		ownerKey: ownerKey,
		KeyBook:  kb,
	}, nil
}

func (s *memStore) Owner() (peer.ID, crypto.PrivKey) {
	return s.ownerID, s.ownerKey
}
