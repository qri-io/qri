package cmd

import (
	"testing"
)

func TestPrintByteInfo(t *testing.T) {
	cases := []struct {
		bytes  int
		expect string
	}{
		{1, "1 byte"},
		{2, "2 bytes"},
		{kilobyte * 4, "4 KBs"},
		{megabyte * 3, "3 MBs"},
		{gigabyte + 1000, "1 GB"},
		{terabyte * 2, "2 TBs"},
		{petabyte * 100, "100 PBs"},
		{exabyte, "1 EB"},
	}

	for i, c := range cases {
		got := printByteInfo(c.bytes)
		if got != c.expect {
			t.Errorf("case %d expect != got: %s != %s", i, c.expect, got)
			continue
		}
	}
}
