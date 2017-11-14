package dataset

import (
	"fmt"
	"testing"
	"time"
)

func TestLicense(t *testing.T) {

}

func CompareLicense(a, b *License) error {
	if a == nil && b == nil {
		return nil
	} else if a == nil && b != nil || a != nil && b == nil {
		return fmt.Errorf("License mistmatch: %s != %s", a, b)
	}

	if a.Type != b.Type {
		return fmt.Errorf("type mismatch: '%s' != '%s'", a.Type, b.Type)
	}

	return nil
}

func TestAccrualDuration(t *testing.T) {
	cases := []struct {
		in     string
		expect time.Duration
	}{
		{"", time.Duration(0)},
		{"R/P10Y", time.Duration(315360000000000000)},
		{"R/P4Y", time.Duration(126144000000000000)},
		{"R/P1Y", time.Duration(31536000000000000)},
		{"R/P2M", time.Duration(25920000000000000)},
		{"R/P3.5D", time.Duration(345600000000000)},
		{"R/P1D", time.Duration(86400000000000)},
		{"R/P2W", time.Duration(1209600000000000)},
		{"R/P6M", time.Duration(15552000000000000)},
		{"R/P2Y", time.Duration(63072000000000000)},
		{"R/P3Y", time.Duration(94608000000000000)},
		{"R/P0.33W", time.Duration(201600000000000)},
		{"R/P0.33M", time.Duration(864000000000000)},
		{"R/PT1S", time.Duration(1000000000)},
		{"R/P1M", time.Duration(2592000000000000)},
		{"R/P3M", time.Duration(4505142857142857)},
		{"R/P0.5M", time.Duration(1296000000000000)},
		{"R/P4M", time.Duration(7884000000000000)},
		{"R/P1W", time.Duration(604800000000000)},
		{"R/PT1H", time.Duration(3600000000000)},
	}

	for i, c := range cases {
		got := AccuralDuration(c.in)
		if got != c.expect {
			t.Errorf("case %d error. expected: %d, got: %d", i, c.expect, got)
		}
	}
}
