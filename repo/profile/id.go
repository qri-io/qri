package profile

import (
	"encoding/json"
	"fmt"

	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// ID is a distinct thing form a peer.ID. They are *NOT* meant to be interchangable
// but the mechanics of peer.ID & profile.ID are exactly the same (for now).
type ID peer.ID

// String implements the stringer interface for ID
func (id ID) String() string {
	return peer.ID(id).Pretty()
}

func (id ID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, id.String())), nil
}

func (id *ID) UnmarshalJSON(data []byte) (err error) {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	fmt.Println(str)
	*id, err = IDB58Decode(str)
	return
}

func (id *ID) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	*id, err = IDB58Decode(str)
	return
}

func (id *ID) MarshalYAML() (interface{}, error) {
	return id.String(), nil
}

// IDB58Decode proxies a lower level API b/c I'm lazy & don't like
func IDB58Decode(proid string) (ID, error) {
	pid, err := peer.IDB58Decode(proid)
	return ID(pid), err
}

// IDB58Decode proxies a lower level API b/c I'm lazy & don't like
func IDB58MustDecode(proid string) ID {
	pid, err := peer.IDB58Decode(proid)
	if err != nil {
		panic(proid + " " + err.Error())
	}
	return ID(pid)
}

// NewB58PeerID creates a peer.ID from a base58-encoded string
func NewB58ID(pid string) (ID, error) {
	id, err := peer.IDB58Decode(pid)
	return ID(id), err
}
