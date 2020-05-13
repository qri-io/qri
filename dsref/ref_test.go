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
			t.Errorf("result mismatch. input:%#v \nwant: '%s'\ngot: '%s'", c.in, c.expect, got)
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
			t.Errorf("result mismatch. input:%#v \nwant: '%s'\ngot: '%s'", c.in, c.expect, got)
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
	}

	for _, c := range cases {
		got := c.in.String()
		if c.expect != got {
			t.Errorf("result mismatch. input:%#v \nwant: '%s'\ngot: '%s'", c.in, c.expect, got)
		}
	}
}
