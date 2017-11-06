package core

import (
	"fmt"
	"net/http"
	"testing"
)

func ListParamsEqual(a, b ListParams) error {
	if a.Limit != b.Limit {
		return fmt.Errorf("ListParams.Limits not equal: '%v' != '%v'", a.Limit, b.Limit)
	}
	if a.Offset != b.Offset {
		return fmt.Errorf("ListParams.Offsets not equal: '%v' != '%v'", a.Offset, b.Offset)
	}
	return nil
}

func TestListParamsFromRequest(t *testing.T) {
	cases := []struct {
		urlStr   string
		res      ListParams
		expected string
	}{
		// [0]
		{"abc.com/123/?pageSize=44&page=22", ListParams{Limit: 43, Offset: 968},
			"ListParams.Limits not equal: '43' != '44'"},
		// [1]
		{"abc.com/123/?pageSize=44&page=22", ListParams{Limit: 44, Offset: 22}, "ListParams.Offsets not equal: '22' != '968'"},
		// [2]
		{"abc.com/123/?pageSize=44&page=22", ListParams{Limit: 44, Offset: 968},
			""},

		// [3]
		{"abc.com/123/?pageSize=-44&page=22", ListParams{Limit: DEFAULT_PAGE_SIZE, Offset: 968},
			"ListParams.Offsets not equal: '968' != '2200'"},
		// [4]
		{"abc.com/123/?pageSize=-44&page=22", ListParams{Limit: DEFAULT_PAGE_SIZE, Offset: 22 * DEFAULT_PAGE_SIZE},
			""},
		// [5]
		{"abc.com/123/?pageSize=44&page=-22", ListParams{Limit: 44, Offset: 968},
			"ListParams.Offsets not equal: '968' != '0'"},
		// [6]
		{"abc.com/123/?pageSize=44&page=-22", ListParams{Limit: 44, Offset: 0},
			""},

		// [7]
		{"abc.com/123/?pageSize=pageSize&page=22", ListParams{Limit: DEFAULT_PAGE_SIZE, Offset: 2200},
			""},
		// [8]
		{"abc.com/123/?pageSize=44&page=abc", ListParams{Limit: 44, Offset: 0},
			""},
		// [9]
		{"abc.com/123/", ListParams{Limit: DEFAULT_PAGE_SIZE, Offset: 0},
			""},
	}

	for i, c := range cases {
		req, err := http.NewRequest("GET", c.urlStr, nil)
		if err != nil {
			t.Errorf("error creating request object: %s", err.Error())
			return
		}
		lp := ListParamsFromRequest(req)
		got := ListParamsEqual(c.res, lp)
		if got != nil && got.Error() != c.expected {
			errorMessage := got.Error()
			t.Errorf("case [%d]: %s", i, errorMessage)
		}
	}
}
