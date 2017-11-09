package porterstemmer

import (
	"bytes"
	"testing"
)

const maxFuzzLen = 6

// Test inputs of English characters less than maxFuzzLen
// Added to help diagnose https://github.com/reiver/go-porterstemmer/issues/4
func TestStemFuzz(t *testing.T) {

	input := []byte{'a'}
	for len(input) < maxFuzzLen {
		// test input

		panicked := false
		func() {
			defer func() { panicked = recover() != nil }()
			StemString(string(input))
		}()
		if panicked {
			t.Errorf("StemString panicked for input '%s'", input)
		}

		// if all z's extend
		if allZs(input) {
			input = bytes.Repeat([]byte{'a'}, len(input)+1)
		} else {
			// increment
			input = incrementBytes(input)
		}
	}
}

func incrementBytes(in []byte) []byte {
	rv := make([]byte, len(in))
	copy(rv, in)
	for i := len(rv) - 1; i >= 0; i-- {
		if rv[i]+1 == '{' {
			rv[i] = 'a'
			continue
		}
		rv[i] = rv[i] + 1
		break

	}
	return rv
}

func allZs(in []byte) bool {
	for _, b := range in {
		if b != 'z' {
			return false
		}
	}
	return true
}
