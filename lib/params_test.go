package lib

import (
	"fmt"
	"net/http"
	"path/filepath"
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

type testStruct struct {
	Name  string
	Path  string `qri:"fspath"`
	Ref   string
	Left  string `qri:"dsrefOrFspath"`
	Right string `qri:"dsrefOrFspath"`
}

func TestNormalizeInputParams(t *testing.T) {
	st := testStruct{
		Name:  "test_data",
		Path:  "testdata/dataset.yml",
		Ref:   "my_peer/my_dataset",
		Left:  "testdata/cities_2/body.csv",
		Right: "my_peer/another_ds",
	}
	normalizeInputParams(&st)

	if st.Name != "test_data" {
		t.Errorf("Name mismatch, expected: test_data, got: %s", st.Name)
	}
	if !filepath.IsAbs(st.Path) {
		t.Errorf("Path mismatch, expected abs path, got: %s", st.Path)
	}
	if st.Ref != "my_peer/my_dataset" {
		t.Errorf("Ref mismatch, expected: my_peer/my_dataset, got: %s", st.Ref)
	}
	if !filepath.IsAbs(st.Left) {
		t.Errorf("Left mismatch, expected abs path, got: %s", st.Left)
	}
	if st.Right != "my_peer/another_ds" {
		t.Errorf("Right mismatch, expected: my_peer/another_ds, got: %s", st.Right)
	}
}
