package profile

import (
	"encoding/json"
	"testing"
)

func TestTypeMarshalJSON(t *testing.T) {
	cases := []struct {
		dt     Type
		expect string
		err    error
	}{
		{TypePeer, "\"peer\"", nil},
		{TypeOrganization, "\"organization\"", nil},
	}

	for i, c := range cases {
		got, err := json.Marshal(c.dt)
		if err != c.err {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
		}
		if string(got) != c.expect {
			t.Errorf("case %d byte mismatch. expected: %s, got: %s", i, c.expect, string(got))
		}
	}
}

func TestTypeUnmarshalJSON(t *testing.T) {
	cases := []struct {
		data []byte
		dt   Type
		err  error
	}{
		{[]byte("[\"peer\"]"), TypePeer, nil},
		{[]byte("[\"user\"]"), TypePeer, nil},
		{[]byte("[\"organization\"]"), TypeOrganization, nil},
	}

	for i, c := range cases {
		var dt []Type
		err := json.Unmarshal(c.data, &dt)
		if err != c.err {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}
		d := dt[0]
		if c.dt != d {
			t.Errorf("case %d byte mismatch. expected: %s, got: %s", i, c.dt, d)
		}
	}
}
