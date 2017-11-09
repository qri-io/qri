package datatypes

import (
	"testing"
)

func TestEachIsType(t *testing.T) {
	cases := []struct {
		typ    Type
		ts     []Type
		expect bool
	}{
		{Integer, []Type{Integer}, true},
		{Integer, []Type{Integer, Float}, false},
		{String, []Type{String, String, String}, true},
		{Float, []Type{String, String, String}, false},
	}

	for i, c := range cases {
		got := EachIsType(c.typ, c.ts)
		if got != c.expect {
			t.Errorf("case %d error: expected: %t, got: %t", i, c.expect, got)
		}
	}
}

func TestEachSame(t *testing.T) {
	cases := []struct {
		ts     []Type
		expect Type
	}{
		{[]Type{Integer}, Integer},
		{[]Type{Integer, Float}, Unknown},
		{[]Type{String, String, String}, String},
	}

	for i, c := range cases {
		got := EachSame(c.ts)
		if got != c.expect {
			t.Errorf("case %d error: expected: %t, got: %t", i, c.expect, got)
		}
	}
}

func TestEachNumeric(t *testing.T) {
	cases := []struct {
		ts     []Type
		expect bool
	}{
		{[]Type{Integer}, true},
		{[]Type{Integer, Float}, true},
		{[]Type{Integer, Float, String}, false},
		{[]Type{String, String, String}, false},
	}

	for i, c := range cases {
		got := EachNumeric(c.ts)
		if got != c.expect {
			t.Errorf("case %d error: expected: %t, got: %t", i, c.expect, got)
		}
	}
}
