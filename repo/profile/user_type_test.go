package profile

import (
	"encoding/json"
	"testing"
)

func TestUserTypeMarshalJSON(t *testing.T) {
	cases := []struct {
		dt     UserType
		expect string
		err    error
	}{
		{UserTypeUser, "\"user\"", nil},
		{UserTypeOrganization, "\"organization\"", nil},
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

func TestUserTypeUnmarshalJSON(t *testing.T) {
	cases := []struct {
		data []byte
		dt   UserType
		err  error
	}{
		{[]byte("[\"user\"]"), UserTypeUser, nil},
		{[]byte("[\"organization\"]"), UserTypeOrganization, nil},
	}

	for i, c := range cases {
		var dt []UserType
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
