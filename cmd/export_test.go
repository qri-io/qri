package cmd

import (
	"testing"
)

func TestFilenameToYAML(t *testing.T) {
	cases := []struct {
		filename, expected string
	}{
		{"", ""},
		{"filename", "filename.yaml"},
		{"filename.json", "filename.yaml"},
		{"filename.", "filename.yaml"},
		{"file.name.json", "file.name.yaml"},
	}

	for i, c := range cases {
		got := FilenameToYAML(c.filename)
		if got != c.expected {
			t.Errorf("case %d (expected != got): '%s' != '%s'", i, c.expected, got)
			continue
		}
	}
}
