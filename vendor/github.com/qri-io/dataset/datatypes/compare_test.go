package datatypes

import (
	"testing"
)

func TestCompareTypeBytes(t *testing.T) {
	cases := []struct {
		a, b   string
		t      Type
		expect int
		err    string
	}{
		{"0", "0", Unknown, 0, "invalid type comparison"},
		{"", "", String, 0, ""},
		{"", "foo", String, -1, ""},
		{"foo", "", String, 1, ""},
		{"foo", "bar", String, 1, ""},
		{"bar", "foo", String, -1, ""},
		{"0", "0", Float, 0, ""},
		{"0", "0", Integer, 0, ""},
	}

	for i, c := range cases {
		got, err := CompareTypeBytes([]byte(c.a), []byte(c.b), c.t)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %d, got: %s", i, c.err, err)
			continue
		}
		if got != c.expect {
			t.Errorf("case %d response mismatch: %d != %d", i, c.expect, got)
			continue
		}
	}
}

func TestCompareIntegerBytes(t *testing.T) {
	cases := []struct {
		a, b   string
		expect int
		err    string
	}{
		{"0", "", 0, "strconv.ParseInt: parsing \"\": invalid syntax"},
		{"", "0", 0, "strconv.ParseInt: parsing \"\": invalid syntax"},
		{"0", "0", 0, ""},
		{"-1", "0", -1, ""},
		{"0", "-1", 1, ""},
	}

	for i, c := range cases {
		got, err := CompareIntegerBytes([]byte(c.a), []byte(c.b))
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}
		if got != c.expect {
			t.Errorf("case %d response mismatch: %d != %d", i, c.expect, got)
			continue
		}
	}
}

func TestCompareFloatBytes(t *testing.T) {
	cases := []struct {
		a, b   string
		expect int
		err    string
	}{
		{"0", "", 0, "strconv.ParseFloat: parsing \"\": invalid syntax"},
		{"", "0", 0, "strconv.ParseFloat: parsing \"\": invalid syntax"},
		{"0", "0", 0, ""},
		{"-1", "0", -1, ""},
		{"0", "-1", 1, ""},
	}

	for i, c := range cases {
		got, err := CompareFloatBytes([]byte(c.a), []byte(c.b))
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %d, got: %s", i, c.err, err)
			continue
		}
		if got != c.expect {
			t.Errorf("case %d response mismatch: %d != %d", i, c.expect, got)
			continue
		}
	}
}
