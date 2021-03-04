package key

import (
	"encoding/json"
	"errors"
	"sync"

	ic "github.com/libp2p/go-libp2p-core/crypto"
)

// Book defines the interface for keybook implementations
// which hold the key information
type Book interface {
	// PubKey stores the public key for a key.ID
	PubKey(ID) ic.PubKey

	// AddPubKey stores the public for a key.ID
	AddPubKey(ID, ic.PubKey) error

	// PrivKey returns the private key for a key.ID, if known
	PrivKey(ID) ic.PrivKey

	// AddPrivKey stores the private key for a key.ID
	AddPrivKey(ID, ic.PrivKey) error

	// IDsWithKeys returns all the key IDs stored in the KeyBook
	IDsWithKeys() []ID
}

type memoryKeyBook struct {
	sync.RWMutex // same lock. wont happen a ton.
	pks          map[ID]ic.PubKey
	sks          map[ID]ic.PrivKey
}

var _ Book = (*memoryKeyBook)(nil)

func newKeyBook() *memoryKeyBook {
	return &memoryKeyBook{
		pks: map[ID]ic.PubKey{},
		sks: map[ID]ic.PrivKey{},
	}
}

// IDsWithKeys returns the list of IDs in the KeyBook
func (mkb *memoryKeyBook) IDsWithKeys() []ID {
	mkb.RLock()
	ps := make([]ID, 0, len(mkb.pks)+len(mkb.sks))
	for p := range mkb.pks {
		ps = append(ps, p)
	}
	for p := range mkb.sks {
		if _, found := mkb.pks[p]; !found {
			ps = append(ps, p)
		}
	}
	mkb.RUnlock()
	return ps
}

// PubKey returns the public key for a given ID if it exists
func (mkb *memoryKeyBook) PubKey(k ID) ic.PubKey {
	mkb.RLock()
	pk := mkb.pks[k]
	mkb.RUnlock()
	// TODO(arqu): we ignore the recovery mechanic to avoid magic
	// behavior in above stores. We should revisit once we work out
	// the broader mechanics of managing keys.
	// pk, _ = p.ExtractPublicKey()
	// if err == nil {
	// 	mkb.Lock()
	// 	mkb.pks[p] = pk
	// 	mkb.Unlock()
	// }
	return pk
}

// AddPubKey inserts a public key for a given ID
func (mkb *memoryKeyBook) AddPubKey(k ID, pk ic.PubKey) error {
	mkb.Lock()
	mkb.pks[k] = pk
	mkb.Unlock()
	return nil
}

// PrivKey returns the private key for a given ID if it exists
func (mkb *memoryKeyBook) PrivKey(k ID) ic.PrivKey {
	mkb.RLock()
	sk := mkb.sks[k]
	mkb.RUnlock()
	return sk
}

// AddPrivKey inserts a private key for a given ID
func (mkb *memoryKeyBook) AddPrivKey(k ID, sk ic.PrivKey) error {
	if sk == nil {
		return errors.New("sk is nil (PrivKey)")
	}

	mkb.Lock()
	mkb.sks[k] = sk
	mkb.Unlock()
	return nil
}

// MarshalJSON implements the JSON marshal interface
func (mkb *memoryKeyBook) MarshalJSON() ([]byte, error) {
	mkb.RLock()
	res := map[string]interface{}{}
	pubKeys := map[string]string{}
	privKeys := map[string]string{}
	for k, v := range mkb.pks {
		byteKey, err := ic.MarshalPublicKey(v)
		if err != nil {
			// skip/don't marshal ill formed keys
			log.Debugf("keybook: failed to marshal key: %q", err.Error())
			continue
		}
		pubKeys[k.Pretty()] = ic.ConfigEncodeKey(byteKey)
	}
	for k, v := range mkb.sks {
		byteKey, err := ic.MarshalPrivateKey(v)
		if err != nil {
			// skip/don't marshal ill formed keys
			log.Debugf("keybook: failed to marshal key: %q", err.Error())
			continue
		}
		privKeys[k.Pretty()] = ic.ConfigEncodeKey(byteKey)
	}

	res["public_keys"] = pubKeys
	res["private_keys"] = privKeys

	mkb.RUnlock()
	return json.Marshal(res)
}

// UnmarshalJSON implements the JSON unmarshal interface
func (mkb *memoryKeyBook) UnmarshalJSON(data []byte) error {
	keyBookJSON := map[string]map[string]string{}
	err := json.Unmarshal(data, &keyBookJSON)
	if err != nil {
		return err
	}
	if pubKeys, ok := keyBookJSON["public_keys"]; ok {
		for k, v := range pubKeys {
			byteKey, err := ic.ConfigDecodeKey(v)
			if err != nil {
				return err
			}
			key, err := ic.UnmarshalPublicKey(byteKey)
			if err != nil {
				return err
			}
			id, err := DecodeID(k)
			if err != nil {
				return err
			}
			err = mkb.AddPubKey(id, key)
			if err != nil {
				return err
			}
		}
	}
	if privKeys, ok := keyBookJSON["private_keys"]; ok {
		for k, v := range privKeys {
			byteKey, err := ic.ConfigDecodeKey(v)
			if err != nil {
				return err
			}
			key, err := ic.UnmarshalPrivateKey(byteKey)
			if err != nil {
				return err
			}
			id, err := DecodeID(k)
			if err != nil {
				return err
			}
			err = mkb.AddPrivKey(id, key)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
