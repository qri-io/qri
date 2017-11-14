package dsfs

import (
	"testing"
)

func TestRefType(t *testing.T) {
	cases := []struct {
		in, typ, out string
	}{
		{"test_name", "name", "test_name"},
		{"QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M", "path", "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json"},
		{"/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M", "path", "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json"},
		{"QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json", "path", "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json"},
	}

	for i, c := range cases {
		gotT, got := RefType(c.in)
		if c.typ != gotT {
			t.Errorf("case %d type mismatch. expected: %s, got: %s", i, gotT, c.typ)
			continue
		}
		if c.out != got {
			t.Errorf("case %d result mismatch. expected: %s, got: %s", i, c.out, got)
			continue
		}
	}
}
