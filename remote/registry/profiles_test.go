package registry

import (
	"encoding/base64"
	"math/rand"
	"testing"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
)

func TestRegisterProfile(t *testing.T) {
	ps := NewMemProfiles()

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

	src = rand.New(rand.NewSource(10000))
	// TODO (b5) - this makes tests slow. use a cache
	key1, _, err := crypto.GenerateEd25519Key(src)
	if err != nil {
		t.Error(err.Error())
		return
	}
	p2, err := ProfileFromPrivateKey(&Profile{Username: "key0"}, key1)
	if err != nil {
		t.Error(err.Error())
		return
	}

	mismatchSig, err := key0.Sign([]byte("bad_data"))
	if err != nil {
		t.Error(err.Error())
		return
	}

	p3, err := ProfileFromPrivateKey(&Profile{Username: "renamed"}, key0)
	if err != nil {
		t.Error(err.Error())
		return
	}

	cases := []struct {
		p   *Profile
		err string
	}{
		{&Profile{Username: "a"}, "profileID is required"},
		{&Profile{ProfileID: p.ProfileID, Username: p.Username, Signature: p.Signature, PublicKey: "bad_data"}, "publickey base64 encoding: illegal base64 data at input byte 3"},
		{&Profile{ProfileID: p.ProfileID, Username: p.Username, Signature: p.Signature, PublicKey: base64.StdEncoding.EncodeToString([]byte("bad_data"))}, "invalid publickey: unexpected EOF"},
		{&Profile{ProfileID: p.ProfileID, Username: p.Username, PublicKey: p.PublicKey, Signature: "bad_data"}, "signature base64 encoding: illegal base64 data at input byte 3"},
		{&Profile{ProfileID: p.ProfileID, Username: p.Username, PublicKey: p.PublicKey, Signature: base64.StdEncoding.EncodeToString([]byte("bad_data"))}, "invalid signature: malformed signature: no header magic"},
		{&Profile{ProfileID: p.ProfileID, Username: p.Username, PublicKey: p.PublicKey, Signature: base64.StdEncoding.EncodeToString(mismatchSig)}, "mismatched signature"},
		{p, ""},
		{p, ""}, // check that peer can double-register their own username without err
		{p2, "username 'key0' is taken"},
		{p3, ""},
	}

	for i, c := range cases {
		err := RegisterProfile(ps, c.p)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
		}
	}

	if err := DeregisterProfile(ps, &Profile{}); err == nil {
		t.Error("invalid profile should error")
	}
	if err := DeregisterProfile(ps, &Profile{ProfileID: p.ProfileID, Username: p.Username, PublicKey: p.PublicKey, Signature: base64.StdEncoding.EncodeToString(mismatchSig)}); err == nil {
		t.Error("unverifiable profile should error")
	}
	if err := DeregisterProfile(ps, p2); err != nil {
		t.Errorf("error deregistering: %s", err.Error())
	}
}

func TestProfilesSortedRange(t *testing.T) {
	ps := NewMemProfiles()

	src := rand.New(rand.NewSource(0))
	usernames := []string{"a", "b", "c"}
	for _, username := range usernames {
		pkey, _, err := crypto.GenerateSecp256k1Key(src)
		if err != nil {
			t.Error(err.Error())
			return
		}
		p, err := ProfileFromPrivateKey(&Profile{Username: username}, pkey)
		if err != nil {
			t.Error(err.Error())
			return
		}

		if err := RegisterProfile(ps, p); err != nil {
			t.Error(err.Error())
			return
		}
	}

	l, err := ps.Len()
	if err != nil {
		t.Fatal(err)
	}

	if l != len(usernames) {
		t.Errorf("expected len to equal username length")
		return
	}

	for iter := 0; iter < 100; iter++ {
		i := 0
		ps.SortedRange(func(key string, p *Profile) (bool, error) {
			if usernames[i] != p.Username {
				t.Errorf("iter: %d sorted index %d mismatch. expected: %s, got: %s", iter, i, usernames[i], p.Username)
				return true, nil
			}
			i++
			return false, nil
		})
		break
	}
}
