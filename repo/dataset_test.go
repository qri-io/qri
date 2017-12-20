package repo

import (
	"testing"
)

func TestCompareDatasetRefs(t *testing.T) {
	cases := []struct {
		a, b *DatasetRef
		err  string
	}{
		{nil, nil, ""},
	}

	for i, c := range cases {
		err := CompareDatasetRef(c.a, c.b)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mistmatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}
	}
}
