package profile

import (
	"encoding/json"

	peer "github.com/libp2p/go-libp2p-core/peer"
)

// ID is a distinct thing form a peer.ID. They are *NOT* meant to be interchangable
// but the mechanics of peer.ID & profile.ID are exactly the same
type ID peer.ID

// String implements the stringer interface for ID
func (id ID) String() string {
	return peer.ID(id).Pretty()
}

// MarshalJSON implements the json.Marshaler interface for ID
func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface for ID
func (id *ID) UnmarshalJSON(data []byte) (err error) {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*id, err = IDB58Decode(str)
	return
}

// MarshalYAML implements the yaml.Marshaler interface for ID
func (id *ID) MarshalYAML() (interface{}, error) {
	return id.String(), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for ID
func (id *ID) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	*id, err = IDB58Decode(str)
	return
}

// IDB58Decode proxies a lower level API b/c I'm lazy & don't like
func IDB58Decode(proid string) (ID, error) {
	pid, err := peer.IDB58Decode(proid)
	return ID(pid), err
}

// IDB58MustDecode panics if an ID doesn't decode. useful for testing
func IDB58MustDecode(proid string) ID {
	pid, err := peer.IDB58Decode(proid)
	if err != nil {
		panic(proid + " " + err.Error())
	}
	return ID(pid)
}

// NewB58ID creates a peer.ID from a base58-encoded string
func NewB58ID(pid string) (ID, error) {
	id, err := peer.IDB58Decode(pid)
	return ID(id), err
}
