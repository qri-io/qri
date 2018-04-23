package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/repo/profile"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestProfileRequestsGet(t *testing.T) {
	cases := []struct {
		in  bool
		res *profile.Profile
		err string
	}{
		// {true, nil, ""},
		// {false, nil, ""},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewProfileRequests(mr, nil)
	for i, c := range cases {
		got := &profile.CodingProfile{}
		err := req.GetProfile(&c.in, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestProfileRequestsSave(t *testing.T) {
	prev := SaveConfig
	SaveConfig = func() error {
		return nil
	}
	defer func() { SaveConfig = prev }()

	cases := []struct {
		p   *profile.CodingProfile
		res *profile.CodingProfile
		err string
	}{
		{nil, nil, "profile required for update"},
		{&profile.CodingProfile{}, nil, ""},
		// TODO - moar tests
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewProfileRequests(mr, nil)
	for i, c := range cases {
		got := &profile.CodingProfile{}
		err := req.SaveProfile(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestProfileRequestsSetProfilePhoto(t *testing.T) {
	prev := SaveConfig
	SaveConfig = func() error {
		return nil
	}
	defer func() { SaveConfig = prev }()

	cases := []struct {
		infile  string
		respath datastore.Key
		err     string
	}{
		{"", datastore.NewKey(""), "file is required"},
		{"testdata/ink_big_photo.jpg", datastore.NewKey(""), "file size too large. max size is 250kb"},
		{"testdata/q_bang.svg", datastore.NewKey(""), "invalid file format. only .jpg images allowed"},
		{"testdata/rico_400x400.jpg", datastore.NewKey("/map/QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1"), ""},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewProfileRequests(mr, nil)
	for i, c := range cases {
		p := &FileParams{}
		if c.infile != "" {
			p.Filename = filepath.Base(c.infile)
			r, err := os.Open(c.infile)
			if err != nil {
				t.Errorf("case %d error opening test file %s: %s ", i, c.infile, err.Error())
				continue
			}
			p.Data = r
		}

		res := &profile.CodingProfile{}
		err := req.SetProfilePhoto(p, res)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err.Error())
			continue
		}

		if !c.respath.Equal(datastore.NewKey(res.Photo)) {
			t.Errorf("case %d profile hash mismatch. expected: %s, got: %s", i, c.respath.String(), res.Photo)
			continue
		}
	}
}

func TestProfileRequestsSetPosterPhoto(t *testing.T) {
	prev := SaveConfig
	SaveConfig = func() error {
		return nil
	}
	defer func() { SaveConfig = prev }()

	cases := []struct {
		infile  string
		respath datastore.Key
		err     string
	}{
		{"", datastore.NewKey(""), "file is required"},
		{"testdata/ink_big_photo.jpg", datastore.NewKey(""), "file size too large. max size is 250kb"},
		{"testdata/q_bang.svg", datastore.NewKey(""), "invalid file format. only .jpg images allowed"},
		{"testdata/rico_poster_1500x500.jpg", datastore.NewKey("/map/QmdJgfxj4rocm88PLeEididS7V2cc9nQosA46RpvAnWvDL"), ""},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewProfileRequests(mr, nil)
	for i, c := range cases {
		p := &FileParams{}
		if c.infile != "" {
			p.Filename = filepath.Base(c.infile)
			r, err := os.Open(c.infile)
			if err != nil {
				t.Errorf("case %d error opening test file %s: %s ", i, c.infile, err.Error())
				continue
			}
			p.Data = r
		}

		res := &profile.CodingProfile{}
		err := req.SetProfilePhoto(p, res)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err.Error())
			continue
		}

		if !c.respath.Equal(datastore.NewKey(res.Photo)) {
			t.Errorf("case %d profile hash mismatch. expected: %s, got: %s", i, c.respath.String(), res.Photo)
			continue
		}
	}
}
