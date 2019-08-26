package registry

import (
	"math/rand"
	"testing"

	"github.com/libp2p/go-libp2p-crypto"
)

func TestProfileValidate(t *testing.T) {
	src := rand.New(rand.NewSource(0))
	key0, _, err := crypto.GenerateSecp256k1Key(src)
	if err != nil {
		t.Error(err.Error())
		return
	}
	p, err := ProfileFromPrivateKey("key0", key0)
	if err != nil {
		t.Error(err.Error())
		return
	}

	cases := []struct {
		p   *Profile
		err string
	}{
		{&Profile{}, "handle is required"},
		{&Profile{Handle: p.Handle}, "profileID is required"},
		{&Profile{Handle: p.Handle, ProfileID: p.ProfileID}, "signature is required"},
		{&Profile{Handle: p.Handle, ProfileID: p.ProfileID, Signature: p.Signature}, "publickey is required"},
		{&Profile{Handle: p.Handle, ProfileID: p.ProfileID, Signature: p.Signature, PublicKey: p.PublicKey}, ""},
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
		handle  string
		privKey crypto.PrivKey
		profile *Profile
		err     string
	}{
		{"handle", key0, &Profile{Handle: "handle"}, ""},
	}

	for i, c := range cases {
		p, err := ProfileFromPrivateKey(c.handle, c.privKey)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d err mismatch. expected: %s, got: %s", i, c.err, err)
			return
		}

		if c.profile != nil {
			if c.profile.Handle != p.Handle {
				t.Errorf("handle mismatch. expected: %s, got: %s", c.handle, c.profile.Handle)
			}
		}
	}
}
