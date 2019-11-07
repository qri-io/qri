package config

import (
	"reflect"
	"testing"
)

func TestStatsValidate(t *testing.T) {
	err := DefaultStats().Validate()
	if err != nil {
		t.Errorf("error validating default stats: %s", err)
	}
}

func TestStatsCopy(t *testing.T) {
	// build off DefaultStats so we can test that the stats Copy
	// actually copies over correctly
	s := DefaultStats()
	cases := []struct {
		stats *Stats
	}{
		{s},
	}
	for i, c := range cases {
		cpy := c.stats.Copy()
		if !reflect.DeepEqual(cpy, c.stats) {
			t.Errorf("Stats Copy test case %v, stats structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.stats)
			continue
		}
	}
}
