package rev

import (
	"fmt"
	"testing"
)

func TestParseRevs(t *testing.T) {
	cases := []struct {
		in  string
		exp []*Rev
		err string
	}{
		{"", []*Rev{}, "unrecognized revision field: "},
		{"body", []*Rev{&Rev{"bd", 1}}, ""},
		{"md", []*Rev{&Rev{"md", 1}}, ""},
	}

	for i, c := range cases {
		got, err := ParseRevs(c.in)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
		}
		if len(got) != len(c.exp) {
			t.Errorf("case %d len mismatch. expected %d, got: %d", i, len(c.exp), len(got))
		}

		for j, exp := range c.exp {
			if err := EnsureRevEqual(exp, got[j]); err != nil {
				t.Errorf("case %d result %d mismatch: %s", i, j, err)
			}
		}
	}
}

func EnsureRevEqual(a, b *Rev) error {
	if a.Field != b.Field {
		return fmt.Errorf("Field: %s != %s", a.Field, b.Field)
	}
	if a.Gen != b.Gen {
		return fmt.Errorf("Gen: %d != %d", a.Gen, b.Gen)
	}
	return nil
}
