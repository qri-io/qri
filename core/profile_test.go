package core

import (
	"testing"

	"github.com/qri-io/qri/repo/profile"
)

func TestProfileRequestsGet(t *testing.T) {
	cases := []struct {
		in  bool
		res *profile.Profile
		err string
	}{
		{true, nil, ""},
		{false, nil, ""},
	}

	mr, _, err := NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewProfileRequests(mr)
	for i, c := range cases {
		got := &profile.Profile{}
		err := req.GetProfile(&c.in, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestProfileRequestsSave(t *testing.T) {
	cases := []struct {
		p   *profile.Profile
		res *profile.Profile
		err string
	}{
		{nil, nil, "profile required for update"},
		// TODO - moar tests
	}

	mr, _, err := NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewProfileRequests(mr)
	for i, c := range cases {
		got := &profile.Profile{}
		err := req.SaveProfile(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}
