package lib

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCursorToParams(t *testing.T) {
	lp := ListParams{
		OrderBy: "created",
		Limit:   10,
		Offset:  20,
	}
	c := cursor{nextPage: &lp}
	params, err := c.ToParams()
	if err != nil {
		t.Fatal(err)
	}
	actual, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	expect := `{"limit":"10","offset":"20","orderby":"created"}`
	if diff := cmp.Diff(expect, string(actual)); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}
