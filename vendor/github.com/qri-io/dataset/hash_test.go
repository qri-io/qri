package dataset

import (
	"testing"
)

func TestHashBytes(t *testing.T) {
	cases := []struct {
		in  []byte
		out string
		err error
	}{
		{[]byte(""), "QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n", nil},
	}

	for i, c := range cases {
		got, err := HashBytes(c.in)
		if err != c.err {
			t.Errorf("case %d error mismatch. expected: %s got: %s", i, c.err, err)
			continue
		}

		if got != c.out {
			t.Errorf("case %d result mismatch. expected: %s got: %s", i, c.out, got)
			continue
		}
	}
}
