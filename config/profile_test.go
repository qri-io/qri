package config

import (
	"testing"
)

func TestProfileDecodeProfile(t *testing.T) {
	p := &Profile{}
	_, err := p.DecodeProfile()
	if err == nil {
		t.Errorf("expected missing ID to error")
	}

	p = &Profile{
		ID:   "QmTwtwLMKHHKCrugNxyAaZ31nhBqRUQVysT2xK911n4m6F",
		Type: "dinosaur",
	}

	_, err = p.DecodeProfile()
	if err == nil {
		t.Errorf("expected invalid type to error")
	}

	p = &Profile{
		ID:      "QmTwtwLMKHHKCrugNxyAaZ31nhBqRUQVysT2xK911n4m6F",
		Poster:  "foo",
		Profile: "bar",
		Thumb:   "baz",
	}

	pro, err := p.DecodeProfile()
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}

	if pro.Poster.String() != "/foo" {
		t.Error("poster mismatch")
	}
	if pro.Profile.String() != "/bar" {
		t.Error("profile mismatch")
	}
	if pro.Thumb.String() != "/baz" {
		t.Error("thumb mismatch")
	}
}

func TestProfileDecodePrivateKey(t *testing.T) {
	missingErr := "missing private key"
	p := &Profile{}
	_, err := p.DecodePrivateKey()
	if err == nil {
		t.Errorf("expected empty private key to err")
	} else if err.Error() != missingErr {
		t.Errorf("error mismatch. expected: %s, got: %s", missingErr, err.Error())
	}

	invalidErr := "decoding private key: illegal base64 data at input byte 4"
	p = &Profile{PrivKey: "invalid"}
	_, err = p.DecodePrivateKey()
	if err == nil {
		t.Errorf("expected empty private key to err")
	} else if err.Error() != invalidErr {
		t.Errorf("error mismatch. expected: %s, got: %s", invalidErr, err.Error())
	}

	// run this test a few times to ensure default profile consistently generates
	// a valid PrivateKey
	for i := 0; i < 10; i++ {
		p = DefaultProfile()
		_, err = p.DecodePrivateKey()
		if err != nil {
			t.Errorf("iter %d unexpected error: %s", i, err.Error())
		}
	}
}

func TestProfileValidate(t *testing.T) {
	err := DefaultProfile().Validate()
	if err != nil {
		t.Errorf("error validating default profile: %s", err)
	}
}
