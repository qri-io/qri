package params

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParamsWith(t *testing.T) {
	expect := List{
		OrderBy: []string{"1", "2", "3"},
		Filter:  []string{"a", "b", "c"},
		Offset:  200,
		Limit:   100,
	}

	got := ListAll.WithFilters("a", "b", "c").WithOrderBy("1", "2", "3").WithOffsetLimit(200, 100)

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestParamsValidate(t *testing.T) {
	good := []List{
		{Limit: -1, Offset: 0},
	}

	for i, c := range good {
		t.Run(fmt.Sprintf("good_case_%d", i), func(t *testing.T) {
			if err := c.Validate(); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}

	bad := []List{
		{Offset: -1},
		{Limit: -2},
	}

	for i, c := range bad {
		t.Run(fmt.Sprintf("bad_case_%d", i), func(t *testing.T) {
			if err := c.Validate(); err == nil {
				t.Errorf("expected error, got none")
			}
		})
	}
}

func TestListParamsAll(t *testing.T) {
	cases := []struct {
		p      List
		expect bool
	}{
		{List{Limit: -1, Offset: 0}, true},
		{List{OrderBy: []string{"date"}, Limit: -1, Offset: 0}, true},
		{List{Filter: []string{"user:noodles"}, Limit: -1, Offset: 0}, true},

		{List{Limit: 0, Offset: 0}, false},
		{List{Limit: -1, Offset: 100000}, false},
		{List{OrderBy: []string{"time"}, Limit: 0, Offset: 0}, false},
	}

	for i, c := range cases {
		got := c.p.All()
		if c.expect != got {
			t.Errorf("case %d result mismatch. want: %t got: %t", i, c.expect, got)
		}
	}
}
