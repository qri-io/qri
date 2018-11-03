package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAbsPath(t *testing.T) {
	tmp, err := filepath.Abs(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		in, out, err string
	}{
		{"", "", ""},
		{"http://example.com/zipfile.zip", "http://example.com/zipfile.zip", ""},
		{"/ipfs", "/ipfs", ""},
		{tmp, tmp, ""},
	}

	for i, c := range cases {
		got := c.in
		err := AbsPath(&got)
		if !(err == nil && c.err == "" || (err != nil && c.err == err.Error())) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
		}
		if got != c.out {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.out, got)
		}
	}
}

func TestReadDatasetFile(t *testing.T) {

}
