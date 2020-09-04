package util

import (
	"fmt"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestPageFromRequest(t *testing.T) {
	cases := []struct {
		description        string
		queries            map[string]int
		expNumber, expSize int
	}{
		{"no page or pageSize params", map[string]int{}, 1, DefaultPageSize},
		{"negative ints for page and pageSize params", map[string]int{"page": -1, "pageSize": -1}, 1, DefaultPageSize},
		{"no pageSize param", map[string]int{"page": 2}, 2, DefaultPageSize},
		{"no page param", map[string]int{"pageSize": 25}, 1, 25},
		{"happy path", map[string]int{"page": 5, "pageSize": 30}, 5, 30},
	}

	for _, c := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		q := r.URL.Query()
		// add query params
		for key, val := range c.queries {
			q.Set(key, strconv.Itoa(val))
		}
		r.URL.RawQuery = q.Encode()

		got := PageFromRequest(r)
		if c.expNumber != got.Number {
			t.Errorf("case '%s' error: number mismatch, expected '%d', got '%d'", c.description, c.expNumber, got.Number)
		}
		if c.expSize != got.Size {
			t.Errorf("case '%s' error: size mismatch, expected '%d', got '%d'", c.description, c.expSize, got.Size)
		}
	}

}

func TestNewPageFromLimitAndOffset(t *testing.T) {
	cases := []struct {
		description                       string
		offset, limit, expNumber, expSize int
	}{
		{"offset and limit 0", 0, 0, 1, DefaultPageSize},
		{"offset and limit negative", -1, -1, 1, DefaultPageSize},
		{"offset and limit happy path", 150, 25, 7, 25},
		{"offset and limit offset not multiple of limit", 90, 25, 4, 25},
		{"offset and limit larger limit then offset", 25, 150, 1, 150},
	}

	for _, c := range cases {
		got := NewPageFromOffsetAndLimit(c.offset, c.limit)
		if c.expNumber != got.Number {
			t.Errorf("case '%s' error: number mismatch, expected '%d', got '%d'", c.description, c.expNumber, got.Number)
		}
		if c.expSize != got.Size {
			t.Errorf("case '%s' error: size mismatch, expected '%d', got '%d'", c.description, c.expSize, got.Size)
		}
	}
}

func TestNextPageExists(t *testing.T) {
	cases := []struct {
		number, size, resultCount int
		expect                    bool
	}{
		{1, 50, 0, true},
		{1, 50, 51, true},
		{1, 50, 50, false},
		{1, 50, 35, false},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("num_%d_size_%d_count_%d=%t", c.number, c.size, c.resultCount, c.expect), func(t *testing.T) {
			p := Page{Number: c.number, Size: c.size, ResultCount: c.resultCount}
			got := p.NextPageExists()
			if c.expect != got {
				t.Errorf("result mismatch. expected: %t got: %t", c.expect, got)
			}
		})
	}
}

func TestPrevPageExists(t *testing.T) {
	for _, num := range []int{2, 3, 4} {
		p := Page{Number: num}
		if !p.PrevPageExists() {
			t.Errorf("expected true for %d value, got false", num)
		}
	}

	for _, num := range []int{-1, 0, -20} {
		p := Page{Number: num}
		if p.PrevPageExists() {
			t.Errorf("expected false for %d value, got true", num)
		}
	}
}
