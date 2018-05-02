package profile

import (
	"github.com/qri-io/qri/config"
	"testing"
)

func TestProfileDecode(t *testing.T) {
	var (
		p   = &Profile{}
		cp  = &config.ProfilePod{}
		err error
	)

	if err = p.Decode(cp); err == nil {
		t.Errorf("expected missing ID to error")
	}

	cp = &config.ProfilePod{}
	cp.ID = "QmTwtwLMKHHKCrugNxyAaZ31nhBqRUQVysT2xK911n4m6F"
	cp.Type = "dinosaur"

	if err = p.Decode(cp); err == nil {
		t.Errorf("expected invalid type to error")
	}

	cp = &config.ProfilePod{}
	cp.ID = "QmTwtwLMKHHKCrugNxyAaZ31nhBqRUQVysT2xK911n4m6F"
	cp.Poster = "foo"
	cp.Photo = "bar"
	cp.Thumb = "baz"

	if err := p.Decode(cp); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}

	if p.Poster.String() != "/foo" {
		t.Error("poster mismatch")
	}
	if p.Photo.String() != "/bar" {
		t.Error("photo mismatch")
	}
	if p.Thumb.String() != "/baz" {
		t.Error("thumb mismatch")
	}
}

func TestProfileEncode(t *testing.T) {
	cp := &config.ProfilePod{
		ID:       "QmTwtwLMKHHKCrugNxyAaZ31nhBqRUQVysT2xK911n4m6F",
		Peername: "test_profile",
	}

	pro := &Profile{}
	if err := pro.Decode(cp); err != nil {
		t.Error(err.Error())
		return
	}
}
