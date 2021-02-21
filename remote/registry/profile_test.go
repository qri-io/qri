package registry

import (
	"math/rand"
	"testing"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
)

func TestProfileValidate(t *testing.T) {
	src := rand.New(rand.NewSource(0))
	key0, _, err := crypto.GenerateSecp256k1Key(src)
	if err != nil {
		t.Error(err.Error())
		return
	}
	p, err := ProfileFromPrivateKey(&Profile{Username: "key0"}, key0)
	if err != nil {
		t.Error(err.Error())
		return
	}

	cases := []struct {
		p   *Profile
		err string
	}{
		{&Profile{}, "username is required"},
		{&Profile{Username: p.Username}, "profileID is required"},
		{&Profile{Username: p.Username, ProfileID: p.ProfileID}, "signature is required"},
		{&Profile{Username: p.Username, ProfileID: p.ProfileID, Signature: p.Signature}, "publickey is required"},
		{&Profile{Username: p.Username, ProfileID: p.ProfileID, Signature: p.Signature, PublicKey: p.PublicKey}, ""},
	}

	for i, c := range cases {
		err := c.p.Validate()
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d err mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestProfileFromPrivateKey(t *testing.T) {
	src := rand.New(rand.NewSource(0))
	key0, _, err := crypto.GenerateSecp256k1Key(src)
	if err != nil {
		t.Error(err.Error())
		return
	}

	cases := []struct {
		Profile *Profile
		privKey crypto.PrivKey
		res     *Profile
		err     string
	}{
		{&Profile{Username: "handle"}, key0, &Profile{Username: "handle"}, ""},
	}

	for i, c := range cases {
		p, err := ProfileFromPrivateKey(c.Profile, c.privKey)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d err mismatch. expected: %s, got: %s", i, c.err, err)
			return
		}

		if c.res != nil {
			if c.res.Username != p.Username {
				t.Errorf("handle mismatch. expected: %s, got: %s", c.res.Username, p.Username)
			}
		}
	}
}
