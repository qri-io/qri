package profile

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	peer "github.com/libp2p/go-libp2p-core/peer"
)

// ID is a distinct thing form a peer.ID. They are *NOT* meant to be interchangable
// but the mechanics of peer.ID & profile.ID are exactly the same
type ID peer.ID

// String converts a profileID to a string for debugging
func (id ID) String() string {
	bytes := []byte(id)
	return fmt.Sprintf("profile.ID{%s}", hex.EncodeToString(bytes))
}

// Encode converts a profileID into a base58 encoded string
func (id ID) Encode() string {
	return peer.ID(id).Pretty()
}

// Empty returns whether the id is empty
func (id ID) Empty() bool {
	return id.Encode() == ""
}

// Validate validates the profileID, returning an error if it is invalid
func (id ID) Validate() error {
	if err := peer.ID(id).Validate(); err != nil {
		return err
	}
	b64str := id.Encode()
	if strings.HasPrefix(b64str, "Qm") {
		return nil
	}
	if strings.HasPrefix(b64str, "9t") {
		return fmt.Errorf("profile.ID invalid, was double encoded as %q. do not pass a base64 encoded string, instead use IDB58Decode(b64encodedID)", b64str)
	}
	return fmt.Errorf("profile.ID invalid, encodes to %q", b64str)
}

// MarshalJSON encodes the ID for json marshalling
func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.Encode())
}

// UnmarshalJSON unmarshals an id from json
func (id *ID) UnmarshalJSON(data []byte) (err error) {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*id, err = IDB58Decode(str)
	return
}

// MarshalYAML encodes the ID for yaml marshalling
func (id *ID) MarshalYAML() (interface{}, error) {
	return id.Encode(), nil
}

// UnmarshalYAML unmarshals an id from yaml
func (id *ID) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	*id, err = IDB58Decode(str)
	return
}

// IDRawByteString constructs an ID from a raw byte string. No base58 decoding happens.
// Should only be used in tests
func IDRawByteString(data string) ID {
	return ID(data)
}

// IDFromPeerID type casts a peer.ID from ipfs into an ID
func IDFromPeerID(pid peer.ID) ID {
	return ID(pid)
}

// IDB58Decode decodes a base58 string into a profile.ID
func IDB58Decode(proid string) (ID, error) {
	pid, err := peer.IDB58Decode(proid)
	return ID(pid), err
}

// IDB58DecodeOrEmpty decodes an ID, or returns an empty ID if decoding fails
func IDB58DecodeOrEmpty(proid string) ID {
	pid, err := peer.IDB58Decode(proid)
	if err != nil {
		pid = ""
	}
	return ID(pid)
}

// IDB58MustDecode panics if an ID doesn't decode. useful for testing
func IDB58MustDecode(proid string) ID {
	pid, err := peer.IDB58Decode(proid)
	if err != nil {
		panic(proid + " " + err.Error())
	}
	return ID(pid)
}

// NewB58ID creates a peer.ID from a base58 encoded string
// DEPRECATED: This function is confusing, its name implies that it is returning
// a base58 string, but actually it takes a base58 string and decodes it.
// Call IDB58Decode instead.
func NewB58ID(pid string) (ID, error) {
	id, err := peer.IDB58Decode(pid)
	return ID(id), err
}
