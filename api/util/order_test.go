package util

import (
	"net/http/httptest"
	"testing"
)

func TestOrderByFromRequest(t *testing.T) {
	cases := []struct {
		description string
		query       string
		validKeys   []string
		expNumber   int
		expOrderBy  string
	}{
		{"no orderBy specified", "", nil, 0, ""},
		{"orderBy with no direction (defaults to desc)", "title", nil, 1, "title,desc"},
		{"invalid orderBy", "title,asc,date", nil, 0, ""},
		{"orderBy with bad parameters", "title,asc, date,desc", nil, 2, "title,asc,date,desc"},
		{"orderBy with multiple parameters", "title,asc,date,desc", nil, 2, "title,asc,date,desc"},
		{"orderBy with some invalid parameters", "title,asc,date,desc", []string{"title"}, 1, "title,asc"},
	}

	for _, c := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		q := r.URL.Query()
		// add query params
		if c.query != "" {
			q.Set("orderBy", c.query)
		}

		r.URL.RawQuery = q.Encode()

		got := OrderByFromRequestWithKeys(r, c.validKeys)
		if c.expNumber != len(got) {
			t.Errorf("case '%s' error: number mismatch, expected '%d', got '%d'", c.description, c.expNumber, len(got))
		}
		if c.expOrderBy != got.String() {
			t.Errorf("case '%s' error: output mismatch, expected '%s', got '%s'", c.description, c.expOrderBy, got.String())
		}
	}

}
