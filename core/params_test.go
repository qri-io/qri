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
		urlStr string
		res    ListParams
	}{
		{"abc.com/123/", ListParams{Limit: DefaultPageSize, Offset: 0}},
		{"abc.com/123/?pageSize=44&page=22", ListParams{Limit: 44, Offset: 924}},
		{"abc.com/123/?pageSize=-44&page=22", ListParams{Limit: DefaultPageSize, Offset: (22 - 1) * DefaultPageSize}},
		{"abc.com/123/?pageSize=44&page=-22", ListParams{Limit: 44, Offset: 0}},
		{"abc.com/123/?pageSize=pageSize&page=22", ListParams{Limit: DefaultPageSize, Offset: (22 - 1) * DefaultPageSize}},
		{"abc.com/123/?pageSize=44&page=abc", ListParams{Limit: 44, Offset: 0}},
	}

	for i, c := range cases {
		req, err := http.NewRequest("GET", c.urlStr, nil)
		if err != nil {
			t.Errorf("error creating request object: %s", err.Error())
			return
		}

		lp := ListParamsFromRequest(req)

		if err := ListParamsEqual(c.res, lp); err != nil {
			t.Errorf("case [%d] error: %s", i, err.Error())
			continue
		}

	}
}

func TestPage(t *testing.T) {
	cases := []struct {
		input  ListParams
		number int
		size   int
	}{
		{ListParams{Limit: 25, Offset: 0}, 1, 25},
		{ListParams{Limit: 25, Offset: 2}, 1, 25},
		{ListParams{Limit: 25, Offset: 24}, 1, 25},
		{ListParams{Limit: 25, Offset: 25}, 2, 25},
		{ListParams{Limit: 25, Offset: 49}, 2, 25},
		{ListParams{Limit: -100, Offset: 50}, 1, 100},
	}
	for i, c := range cases {
		p := c.input.Page()
		if !(p.Number == c.number && p.Size == c.size) {
			t.Errorf("case %d error mismatch: expected: (%d, %d), got: (%d, %d) for Page.Number, Page.Size", i, c.number, c.size, p.Number, p.Size)
			continue
		}
	}
}
