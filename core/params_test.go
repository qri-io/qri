package core

import (
	"fmt"
	"net/http"
	"testing"
)

func ListParamsEqual(a, b ListParams) error {
	if a.Limit != b.Limit {
		return fmt.Errorf("ListParams.Limit fields not equal: '%d' != '%d'", a.Limit, b.Limit)
	}
	if a.Offset != b.Offset {
		return fmt.Errorf("ListParams.Offset fields not equal: '%d' != '%d'", a.Offset, b.Offset)
	}
	return nil
}

func TestListParamsFromRequest(t *testing.T) {
	cases := []struct {
		urlStr   string
		res      ListParams
		expected string
	}{
		{"abc.com/123/?pageSize=44&page=22",
			ListParams{Limit: 43, Offset: 968},
			"ListParams.Limit fields not equal: '43' != '44'"},
		{"abc.com/123/?pageSize=44&page=22",
			ListParams{Limit: 44, Offset: 22},
			"ListParams.Offset fields not equal: '22' != '968'"},
		{"abc.com/123/?pageSize=44&page=22",
			ListParams{Limit: 44, Offset: 968},
			""},

		{"abc.com/123/?pageSize=-44&page=22",
			ListParams{Limit: DEFAULT_PAGE_SIZE, Offset: 968},
			"ListParams.Offset fields not equal: '968' != '2200'"},
		{"abc.com/123/?pageSize=-44&page=22",
			ListParams{Limit: DEFAULT_PAGE_SIZE, Offset: 22 * DEFAULT_PAGE_SIZE},
			""},
		{"abc.com/123/?pageSize=44&page=-22",
			ListParams{Limit: 44, Offset: 968},
			"ListParams.Offset fields not equal: '968' != '0'"},
		{"abc.com/123/?pageSize=44&page=-22",
			ListParams{Limit: 44, Offset: 0},
			""},

		{"abc.com/123/?pageSize=pageSize&page=22",
			ListParams{Limit: DEFAULT_PAGE_SIZE, Offset: 2200},
			""},
		{"abc.com/123/?pageSize=44&page=abc",
			ListParams{Limit: 44, Offset: 0},
			""},
		{"abc.com/123/",
			ListParams{Limit: DEFAULT_PAGE_SIZE, Offset: 0},
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
