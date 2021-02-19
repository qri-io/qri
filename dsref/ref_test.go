package dsref

import (
	"testing"
)

func TestRefAlias(t *testing.T) {
	cases := []struct {
		in     Ref
		expect string
	}{
		{Ref{}, ""},
		{Ref{Username: "a", Name: "b"}, "a/b"},
		{Ref{Username: "a", Name: "b", Path: "foo"}, "a/b"},
	}

	for _, c := range cases {
		got := c.in.Alias()
		if c.expect != got {
			t.Errorf("result mismatch. input:%#v \nwant: %q\ngot: %q", c.in, c.expect, got)
		}
	}
}

func TestRefHuman(t *testing.T) {
	cases := []struct {
		in     Ref
		expect string
	}{
		{Ref{}, ""},
		{Ref{Username: "a", Name: "b"}, "a/b"},
		{Ref{Username: "a", Name: "b", Path: "foo"}, "a/b"},
	}

	for _, c := range cases {
		got := c.in.Human()
		if c.expect != got {
			t.Errorf("result mismatch. input:%#v \nwant: %q\ngot: %q", c.in, c.expect, got)
		}
	}
}

func TestRefString(t *testing.T) {
	cases := []struct {
		in     Ref
		expect string
	}{
		{Ref{}, ""},
		{Ref{Username: "a", Name: "b"}, "a/b"},
		{Ref{Username: "a", Name: "b"}, "a/b"},
		{Ref{Username: "a", Name: "b", Path: "/foo"}, "a/b@/foo"},
		{Ref{Username: "a", Name: "b", InitID: "initid", Path: "/foo"}, "a/b@initid/foo"},
	}

	for _, c := range cases {
		got := c.in.String()
		if c.expect != got {
			t.Errorf("result mismatch. input:%#v \nwant: %q\ngot: %q", c.in, c.expect, got)
		}
	}
}

func TestRefLegacyProfileIDString(t *testing.T) {
	cases := []struct {
		in     Ref
		expect string
	}{
		{Ref{}, ""},
		{Ref{Username: "a", Name: "b"}, "a/b"},
		{Ref{Username: "a", Name: "b"}, "a/b"},
		{Ref{Username: "a", Name: "b", Path: "/foo"}, "a/b@/foo"},
		{Ref{Username: "a", Name: "b", ProfileID: "ProfileID", Path: "/foo"}, "a/b@ProfileID/foo"},
	}

	for _, c := range cases {
		got := c.in.LegacyProfileIDString()
		if c.expect != got {
			t.Errorf("result mismatch. input:%#v \nwant: %q\ngot: %q", c.in, c.expect, got)
		}
	}
}

func TestRefComplete(t *testing.T) {
	compl := Ref{
		InitID:    "an init id",
		Username:  "some username",
		ProfileID: "hey look a profile ID",
		Name:      "a username. who knows if this is valid",
		Path:      "a path",
	}
	if !compl.Complete() {
		t.Errorf("expected isComplete to return true when all fields are populated")
	}

	bad := []Ref{
		{
			InitID: "an init id",
		},
		{
			InitID:   "an init id",
			Username: "some username",
		},
		{
			InitID:    "an init id",
			Username:  "some username",
			ProfileID: "hey look a profile ID",
		},
		{
			InitID:    "an init id",
			Username:  "some username",
			ProfileID: "hey look a profile ID",
			Name:      "a username. who knows if this is valid",
		},
	}
	for _, ref := range bad {
		if ref.Complete() {
			t.Errorf("expected %s to return false for complete", ref)
		}
	}
}
