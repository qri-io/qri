package test

import (
	"encoding/base64"
	"testing"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qri/auth/key"
)

func TestFixtureKeys(t *testing.T) {
	for i, e := range encoded {
		data, err := base64.StdEncoding.DecodeString(e.B64PrivKey)
		if err != nil {
			t.Errorf("key index %d decoding error: %s", i, err)
			continue
		}
		pk, err := crypto.UnmarshalPrivateKey(data)
		if err != nil {
			t.Errorf("key index %d unmarshaling error: %s", i, err)
			continue
		}

		str := pk.Type().String()
		if str != "RSA" && str != "Ed25519" {
			t.Errorf("key index %d unexpected privKey type: %q", i, str)
		}

		str, err = key.IDFromPrivKey(pk)
		if err != nil {
			t.Errorf("key index %d unexpected error generating ID: %s", i, err)
			continue
		}

		if str != e.B58PeerID {
			t.Errorf("key index mismatch.\ncalculated: %q\nrecorded: %q", str, e.B58PeerID)
		}
	}
}
