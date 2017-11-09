package datatypes

import (
	"errors"
	"testing"
	"time"
)

// TODO
func TestParseAny(t *testing.T) {
	cases := []struct {
		input  []byte
		expect interface{}
		err    error
	}{}
	for i, c := range cases {
		value, got := ParseAny(c.input)
		if value != c.expect {
			t.Errorf("case %d value mismatch. expected: %s, got: %s", i, c.expect, value)
		}
		if c.err != got {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, got)
		}
	}
}

func TestParseString(t *testing.T) {
	cases := []struct {
		input  []byte
		expect string
		err    error
	}{
		{[]byte("foo"), "foo", nil},
	}
	for i, c := range cases {
		value, got := ParseString(c.input)
		if value != c.expect {
			t.Errorf("case %d value mismatch. expected: %s, got: %s", i, c.expect, value)
		}
		if c.err != got {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, got)
		}
	}
}

func TestParseFloat(t *testing.T) {
	cases := []struct {
		input  []byte
		expect float64
		err    error
	}{
		{[]byte("1234567890"), float64(1234567890), nil},
		{[]byte("12345.67890"), float64(12345.67890), nil},
		{[]byte("-12345.67890"), float64(-12345.67890), nil},
	}
	for i, c := range cases {
		value, got := ParseFloat(c.input)
		if value != c.expect {
			t.Errorf("case %d value mismatch. expected: %s, got: %s", i, c.expect, value)
		}
		if c.err != got {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, got)
		}
	}
}

func TestParseInteger(t *testing.T) {
	cases := []struct {
		input  []byte
		expect int64
		err    error
	}{
		{[]byte("1234567890"), int64(1234567890), nil},
		{[]byte("12345.67890"), 0, errors.New(`strconv.ParseInt: parsing "12345.67890": invalid syntax`)},
		{[]byte("-12345.67890"), 0, errors.New(`strconv.ParseInt: parsing "-12345.67890": invalid syntax`)},
	}
	for i, c := range cases {
		value, got := ParseInteger(c.input)
		if value != c.expect {
			t.Errorf("case %d value mismatch. expected: %s, got: %s", i, c.expect, value)
		}
		if got != nil {
			if c.err != nil && got.Error() != c.err.Error() {
				t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, got)
			}
		}
	}
}

func TestParseBoolean(t *testing.T) {
	cases := []struct {
		input  []byte
		expect bool
		err    error
	}{}
	for i, c := range cases {
		value, got := ParseBoolean(c.input)
		if value != c.expect {
			t.Errorf("case %d value mismatch. expected: %s, got: %s", i, c.expect, value)
		}
		if c.err != got {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, got)
		}
	}
}

// TODO
func TestParseObject(t *testing.T) {
	cases := []struct {
		input  []byte
		expect map[string]interface{}
		err    error
	}{}
	for i, c := range cases {
		_, got := ParseObject(c.input)
		// if value != c.expect {
		// 	t.Errorf("case %d value mismatch. expected: %s, got: %s", i, c.expect, value)
		// }
		if c.err != got {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, got)
		}
	}
}

// TODO
func TestParseArray(t *testing.T) {
	cases := []struct {
		input  []byte
		expect []interface{}
		err    error
	}{}
	for i, c := range cases {
		_, got := ParseArray(c.input)
		// if value != c.expect {
		// 	t.Errorf("case %d value mismatch. expected: %s, got: %s", i, c.expect, value)
		// }
		if c.err != got {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, got)
		}
	}
}

// TODO
func TestParseDate(t *testing.T) {
	cases := []struct {
		input  []byte
		expect *time.Time
		err    error
	}{}
	for i, c := range cases {
		value, got := ParseDate(c.input)
		if value.String() != c.expect.String() {
			t.Errorf("case %d value mismatch. expected: %s, got: %s", i, c.expect, value)
		}
		if c.err != got {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, got)
		}
	}
}

func TestParseUrl(t *testing.T) {
	cases := []struct {
		input  string
		expect string
		err    error
	}{
		{"apple.com", "apple.com", nil},
		{"http://qri.io", "http://qri.io", nil},
		{"https://beastmo.de", "https://beastmo.de", nil},
		{"https://beastmo.de/this/path", "https://beastmo.de/this/path", nil},
		{"https://beastmo.de/this/path?input=blah", "https://beastmo.de/this/path?input=blah", nil},
		{"https://beastmo.de/this/path?input=blah#fragment", "https://beastmo.de/this/path?input=blah#fragment", nil},
		{"https://beastmo.de/this/path?input=blah#bad fragment", "https://beastmo.de/this/path?input=blah#bad%20fragment", nil},
	}
	for i, c := range cases {
		value, got := ParseUrl([]byte(c.input))
		if value.String() != c.expect {
			t.Errorf("case %d value mismatch. expected: %s, got: %s", i, c.expect, value.String())
		}
		if got != nil {
			if c.err != nil && got.Error() != c.err.Error() {
				t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, got)
			}
		}
	}
}
