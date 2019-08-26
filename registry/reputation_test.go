package registry

import (
	"testing"
)

func TestReputationValidate(t *testing.T) {
	cases := []struct {
		r   *Reputation
		err string
	}{
		{&Reputation{}, "profileID is required"},
		{&Reputation{ProfileID: "my_id"}, ""},
		{&Reputation{ProfileID: "my_id", Rep: 0}, ""},
	}

	for i, c := range cases {
		err := c.r.Validate()

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d err mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}
	}

}

func TestReputationReputation(t *testing.T) {
	cases := []struct {
		r      *Reputation
		expect int
	}{
		{&Reputation{}, 0},
		{&Reputation{Rep: 0}, 0},
		{&Reputation{Rep: 10}, 10},
		{&Reputation{Rep: -1}, -1},
	}
	for i, c := range cases {
		rep := c.r.Reputation()

		if c.expect != rep {
			t.Errorf("case %d reputation mismatch. expected: %d, got: %d", i, c.expect, rep)
		}
	}
}

func TestSetReputation(t *testing.T) {
	cases := []struct {
		r   *Reputation
		set int
	}{
		{&Reputation{}, 10},
		{&Reputation{Rep: -1}, 0},
		{&Reputation{Rep: 5}, 5},
		{&Reputation{Rep: 1}, -1},
	}
	for i, c := range cases {
		c.r.SetReputation(c.set)

		if c.r.Reputation() != c.set {
			t.Errorf("case %d set reputation mismatch. expected: %d, got: %d", i, c.set, c.r.Reputation())
		}
	}
}
