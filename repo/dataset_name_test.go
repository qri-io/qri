package repo

import (
	"testing"
)

func TestCoerceDatasetName(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"filename.csv", "filename"},
		{"variable-name", "variable_name"},
		{"space name", "space_name"},
		{"CAPSNAME", "capsname"},
	}

	for i, c := range cases {
		if got := CoerceDatasetName(c.in); got != c.out {
			t.Errorf("case %d mismatch. expected: '%s', got: '%s'", i, c.out, got)
		}
	}
}

func TestValidDatasetName(t *testing.T) {
	cases := []struct {
		in  string
		out bool
	}{
		{"space name", false},
		{"hyphen-name", false},
		{"dot.name", false},
		{"/slash/name", false},
		{"CAPSNAME", false},
	}

	for i, c := range cases {
		if got := ValidDatasetName(c.in); got != c.out {
			t.Errorf("case %d mismatch. %t != %t", i, c.out, got)
		}
	}
}
