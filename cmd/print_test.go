package cmd

import (
	"runtime"
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

func TestDoesCommandExist(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if doesCommandExist("ls") == false {
			t.Error("ls command does not exist!")
		}
		if doesCommandExist("ls111") == true {
			t.Error("ls111 command should not exist!")
		}
	}
}
