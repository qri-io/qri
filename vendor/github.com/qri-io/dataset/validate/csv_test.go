package validate

import (
	"strings"
	"testing"
)

func TestCheckCsvRowLengths(t *testing.T) {
	cases := []struct {
		input string
		err   string
	}{
		{rawText1, ""},
		{rawText2, ""},
		{rawText2b, ""},
		{rawText3, ""}, //Note: since there are no commas this should pass
		{rawText4, "error: inconsistent column length on line 4 of length 2 (rather than 1). ensure all csv columns same length"},
	}

	for i, c := range cases {
		r := strings.NewReader(c.input)
		err := CheckCsvRowLengths(r)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case [%d] error mismatch. expected: '%s', got: '%s'", i, c.err, err)
		}
	}
}
