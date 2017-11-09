package dataset

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/jbenet/go-base58"
	"github.com/multiformats/go-multihash"
)

// JSONHash calculates the hash of a json.Marshaler
// It's important to note that this is *NOT* the same as an IPFS hash,
// These hash functions should be used for other things like
// checksumming, in-memory content-addressing, etc.
func JSONHash(m json.Marshaler) (hash string, err error) {
	// marshal to cannoncical JSON representation
	data, err := m.MarshalJSON()
	if err != nil {
		return
	}
	return HashBytes(data)
}

// HashBytes generates the base-58 encoded SHA-256 hash of a byte slice
// It's important to note that this is *NOT* the same as an IPFS hash,
// These hash functions should be used for other things like
// checksumming, in-memory content-addressing, etc.
func HashBytes(data []byte) (hash string, err error) {
	h := sha256.New()

	if _, err = h.Write(data); err != nil {
		return
	}

	mhBuf, err := multihash.Encode(h.Sum(nil), multihash.SHA2_256)
	if err != nil {
		err = fmt.Errorf("error allocating multihash buffer: %s", err.Error())
		return
	}

	hash = base58.Encode(mhBuf)
	return
}
