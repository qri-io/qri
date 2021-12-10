package params

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestListParamsFromRequest(t *testing.T) {
	// should work for nil list params & request with no list information
	r, err := http.NewRequest("POST", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	// should not error if there is an already populated params.List and no list
	// information in the request
	expect := List{Offset: 10}
	got, err := List{Offset: 10}.ListParamsFromRequest(r)
	if err != nil {
		t.Fatalf("ListParamsFromRequest should not error if there is no list information in the request: %s", err)
	}
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("params.List mismatch (+want,-got):\n%s", diff)
	}

	// add list queries to the request
	q := r.URL.Query()
	q.Add("offset", "10")
	q.Add("limit", "10")
	q.Add("filter", "username:peer")
	q.Add("orderby", "+name,-updated")
	r.URL.RawQuery = q.Encode()

	got = List{}
	expect = List{
		Offset:  10,
		Limit:   10,
		Filter:  []string{"username:peer"},
		OrderBy: OrderBy{{Key: "name", Direction: OrderASC}, {Key: "updated", Direction: OrderDESC}},
	}
	got, err = got.ListParamsFromRequest(r)
	if err != nil {
		t.Fatalf("ListParamsFromRequest unexpected error: %s", err)
	}
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("params.List mismatch (+want,-got):\n%s", diff)
	}

	// should error if list params are not empty & there is list info in the request
	got = List{Filter: []string{"name:test"}}
	got, err = got.ListParamsFromRequest(r)
	if !errors.Is(err, ErrListParamsNotEmpty) {
		t.Errorf("expected error to be %q, got %s", ErrListParamsNotEmpty, err)
	}
}

func TestIsEmpty(t *testing.T) {
	cases := []struct {
		description string
		lp          List
		expect      bool
	}{
		{"empty list params", List{}, true},
		{"each field empty", List{Offset: 0, Limit: 0, Filter: []string{}, OrderBy: OrderBy{}}, true},
		{"has offset", List{Offset: 1}, false},
		{"has limit", List{Limit: 1}, false},
		{"has filter", List{Filter: []string{"public:true"}}, false},
		{"has orderby", List{OrderBy: OrderBy{{Key: "", Direction: ""}}}, false},
	}
	for _, c := range cases {
		got := c.lp.IsEmpty()
		if got != c.expect {
			fmt.Errorf("error in case %q, expected %t, got %t", c.description, c.expect, got)
		}
	}
}

func TestParamsWith(t *testing.T) {
	expect := List{
		OrderBy: OrderBy{{Key: "1", Direction: OrderASC}, {Key: "2", Direction: OrderASC}, {Key: "3", Direction: OrderASC}},
		Filter:  []string{"a", "b", "c"},
		Offset:  200,
		Limit:   100,
	}

	got := ListAll.WithFilters("a", "b", "c").WithOrderBy("1,2,3").WithOffsetLimit(200, 100)

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
		{List{OrderBy: OrderBy{{Key: "date", Direction: OrderDESC}}, Limit: -1, Offset: 0}, true},
		{List{Filter: []string{"user:noodles"}, Limit: -1, Offset: 0}, true},

		{List{Limit: 0, Offset: 0}, false},
		{List{Limit: -1, Offset: 100000}, false},
		{List{OrderBy: OrderBy{{Key: "time", Direction: OrderASC}}, Limit: 0, Offset: 0}, false},
	}

	for i, c := range cases {
		got := c.p.All()
		if c.expect != got {
			t.Errorf("case %d result mismatch. want: %t got: %t", i, c.expect, got)
		}
	}
}
