package test

import (
	"testing"

	"github.com/qri-io/qri/repo/profile"
)

func testProfileStore(t *testing.T, rmf RepoMakerFunc) {
	r, cleanup := rmf(t)
	defer cleanup()

	ps := r.Profiles()

	if err := ps.PutProfile(&profile.Profile{}); err == nil {
		t.Error("expected PutProfile to require an ID field")
	}

	id, err := profile.IDB58Decode("QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt")
	if err != nil {
		t.Fatal(err)
	}
	if err := ps.PutProfile(&profile.Profile{ID: id, Peername: "uniq"}); err != nil {
		t.Errorf("PutProfile err: %s", err)
	}

	p, err := ps.GetProfile(id)
	if err != nil {
		t.Errorf("GetProfile: %s", err)
	}

	// TODO - write a CompareProfiles function in profile package
	if p.Peername != "uniq" {
		t.Errorf("GetProfile returned the wrong profile")
	}

	if err := ps.DeleteProfile(id); err != nil {
		t.Errorf("DeleteProfile err: %s", err)
	}

}
