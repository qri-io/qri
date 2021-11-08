package lib

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestToJSON(t *testing.T) {
	params := ListParams{
		OrderBy: "created",
		Limit:   10,
		Offset:  20,
	}
	c := cursor{nextPage: &params}
	actual, err := c.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	expect := `{"orderBy":"created","limit":10,"offset":20}`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

func TestToQueryParams(t *testing.T) {
	params := ListParams{
		OrderBy: "created",
		Limit:   10,
		Offset:  20,
	}
	c := cursor{nextPage: &params}
	actual, err := c.ToQueryParams()
	if err != nil {
		t.Fatal(err)
	}
	expect := "?orderby=created&limit=10&offset=20"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}
