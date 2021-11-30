package util

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWritePageResponse(t *testing.T) {
	cases := []struct {
		page   Page
		expect string
	}{
		{
			Page{Number: 1, Size: DefaultPageSize},
			`{"data":"data","meta":{"code":200},"pagination":{"page":1,"pageSize":50,"nextUrl":"https://example.com?page=2","prevUrl":""}}`,
		},
		{
			Page{Number: 1, Size: DefaultPageSize, ResultCount: 200},
			`{"data":"data","meta":{"code":200},"pagination":{"page":1,"pageSize":50,"resultCount":200,"nextUrl":"https://example.com?page=2","prevUrl":""}}`,
		},
		{
			Page{Number: 2, Size: DefaultPageSize, ResultCount: 100},
			`{"data":"data","meta":{"code":200},"pagination":{"page":2,"pageSize":50,"resultCount":100,"nextUrl":"","prevUrl":"https://example.com?page=1"}}`,
		},
		{
			Page{Number: 2, Size: DefaultPageSize, ResultCount: 200},
			`{"data":"data","meta":{"code":200},"pagination":{"page":2,"pageSize":50,"resultCount":200,"nextUrl":"https://example.com?page=3","prevUrl":"https://example.com?page=1"}}`,
		},
		{
			Page{Number: 2, Size: 5, ResultCount: 200},
			`{"data":"data","meta":{"code":200},"pagination":{"page":2,"pageSize":5,"resultCount":200,"nextUrl":"https://example.com?page=3\u0026pageSize=5","prevUrl":"https://example.com?page=1\u0026pageSize=5"}}`,
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			rr := httptest.NewRecorder()

			req, err := http.NewRequest("GET", "https://example.com", nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := WritePageResponse(rr, "data", req, c.page); err != nil {
				t.Fatal(err)
			}

			got := rr.Body.String()

			if c.expect != got {
				t.Errorf("result mismatch. expected:\n%s\ngot:\n%s", c.expect, got)
			}

		})
	}
}

func TestWriteResponseWithNextPage(t *testing.T) {
	rr := httptest.NewRecorder()

	if err := WriteResponseWithNextPage(rr, "data", "/next", map[string]string{"start": "5"}); err != nil {
		t.Fatal(err)
	}

	actual := rr.Body.String()
	expect := `{"data":"data","meta":{"code":200},"nextPage":{"url":"/next","params":{"start":"5"}}}`
	if expect != actual {
		t.Errorf("result mismatch. expected:\n%s\ngot:\n%s", expect, actual)
	}
}
