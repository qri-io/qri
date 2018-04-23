package profile

import (
	"github.com/qri-io/qri/config"
	"testing"
)

func TestProfileDecode(t *testing.T) {
	var (
		p   = &Profile{}
		sp  = &CodingProfile{}
		err error
	)

	if err = p.Decode(sp); err == nil {
		t.Errorf("expected missing ID to error")
	}

	sp = &CodingProfile{}
	sp.ID = "QmTwtwLMKHHKCrugNxyAaZ31nhBqRUQVysT2xK911n4m6F"
	sp.Type = "dinosaur"

	if err = p.Decode(sp); err == nil {
		t.Errorf("expected invalid type to error")
	}

	sp = &CodingProfile{}
	sp.ID = "QmTwtwLMKHHKCrugNxyAaZ31nhBqRUQVysT2xK911n4m6F"
	sp.Poster = "foo"
	sp.Photo = "bar"
	sp.Thumb = "baz"

	if err := p.Decode(sp); err != nil {
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
	sp := NewCodingProfile(&config.Profile{
		ID:       "QmTwtwLMKHHKCrugNxyAaZ31nhBqRUQVysT2xK911n4m6F",
		Peername: "test_profile",
	})

	pro := &Profile{}
	if err := pro.Decode(sp); err != nil {
		t.Error(err.Error())
		return
	}
}
