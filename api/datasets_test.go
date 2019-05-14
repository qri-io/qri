package api

import (
	"net/http"
	"strconv"
	"testing"
)

func TestMakeGetBodyParams(t *testing.T) {
	cases := []struct {
		description                   string
		download                      bool
		format                        string
		page, pageSize, offset, limit int
		expOffset, expLimit           int
		all, readOnly                 bool
		expAll                        bool
		expectedErr                   string
	}{
		{"download not set, format set",
			false,
			"foo",
			-10, -10, -10, -10,
			-10, -10,
			false,
			false,
			false,
			"the format must be json if used without the download parameter",
		},
		{
			"page and pageSize",
			false,
			"",
			1, 25, -10, -10,
			0, 25,
			false,
			false,
			false,
			"",
		},
		{
			"page and pageSize, and offset overrides",
			false,
			"",
			1, 40, 0, -10,
			0, 100,
			false,
			false,
			false,
			"",
		},
		{
			"page and pageSize, and limit overrides",
			false,
			"",
			1, 40, -10, 20,
			0, 20,
			false,
			false,
			false,
			"",
		},
		{
			"request all",
			false,
			"",
			-10, -10, -10, -10,
			0, 100,
			true,
			false,
			true,
			"",
		},
		{
			"request all via offset and limit",
			false,
			"",
			-10, -10, 0, -1,
			0, -1,
			false,
			false,
			true,
			"",
		},
		{
			"readOnly should ignore limit and offset",
			false,
			"",
			3, 30, 0, 10,
			60, 30,
			false,
			true,
			false,
			"",
		},
		{
			"readOnly should override all",
			false,
			"",
			3, 30, -10, -10,
			60, 30,
			true,
			true,
			false,
			"",
		},
	}

	for _, c := range cases {
		// create new request
		// get the query from the request
		//
		r, err := http.NewRequest("GET", "/body", nil)
		if err != nil {
			t.Fatalf("case '%s', error creating request: %s", c.description, err)
			return
		}
		q := r.URL.Query()
		q.Set("download", strconv.FormatBool(c.download))
		if c.format != "" {
			q.Set("format", c.format)
		}
		if c.page > -10 {
			q.Set("page", strconv.Itoa(c.page))
		}
		if c.pageSize > -10 {
			q.Set("pageSize", strconv.Itoa(c.pageSize))
		}
		if c.offset > -10 {
			q.Set("offset", strconv.Itoa(c.offset))
		}
		if c.limit > -10 {
			q.Set("limit", strconv.Itoa(c.limit))
		}
		q.Set("all", strconv.FormatBool(c.all))
		r.URL.RawQuery = q.Encode()

		p, err := makeGetBodyParams(r, c.readOnly, "/body/path")
		if err != nil || c.expectedErr != "" {
			if err != nil && err.Error() != c.expectedErr || (err == nil && c.expectedErr == "") {
				t.Errorf("case '%s' error mismatch. Expected: '%s', Got: '%s'", c.description, c.expectedErr, err)
			}
			continue
		}
		if p.Offset != c.expOffset {
			t.Errorf("case '%s', offset mismatch. Expected: %d, Got: %d", c.description, c.expOffset, p.Offset)
		}
		if p.Limit != c.expLimit {
			t.Errorf("case '%s', limit mismatch. Expected: %d, Got: %d", c.description, c.expLimit, p.Limit)
		}
		if p.All != c.expAll {
			t.Errorf("case '%s', all mismatch. Expected: %t, Got: %t", c.description, c.expAll, p.All)
		}
	}
}
