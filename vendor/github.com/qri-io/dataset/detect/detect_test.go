package detect

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/qri-io/dataset"
)

func TestFromFile(t *testing.T) {
	cases := []struct {
		inpath, dspath string
		err            error
	}{
		{"testdata/hours-with-header.csv", "testdata/hours-with-header.resource.json", nil},
		{"testdata/hours.csv", "testdata/hours.resource.json", nil},
		{"testdata/spelling.csv", "testdata/spelling.resource.json", nil},
		{"testdata/daily_wind_2011.csv", "testdata/daily_wind_2011.resource.json", nil},
	}

	for i, c := range cases {
		data, err := ioutil.ReadFile(c.dspath)
		if err != nil {
			t.Error(err)
			return
		}

		expect := &dataset.Structure{}
		if err := json.Unmarshal(data, expect); err != nil {
			t.Error(err)
			return
		}

		ds, err := FromFile(c.inpath)
		if c.err != err {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			return
		}

		// if ds.Name != expect.Name {
		// 	t.Errorf("case %d name mismatch. expected '%s', got '%s'", i, expect.Name, ds.Name)
		// }

		if expect.Format != ds.Format {
			t.Errorf("case %d format mismatch. expected '%s', got '%s'", i, expect.Format, ds.Format)
		}

		// if expect.File != ds.File {
		// 	t.Errorf("case %d file mismatch. expected '%s', got '%s'", i, expect.File, ds.File)
		// }

		if len(expect.Schema.Fields) != len(ds.Schema.Fields) {
			t.Errorf("case %d field length mismatch. expected: %d, got: %d", i, len(expect.Schema.Fields), len(ds.Schema.Fields))
			return
		}

		for j, f := range expect.Schema.Fields {
			if f.Type != ds.Schema.Fields[j].Type {
				t.Errorf("case %d field %d:%s type mismatch. expected: %s, got: %s", i, j, f.Name, f.Type, ds.Schema.Fields[j].Type)
			}
			if f.Name != ds.Schema.Fields[j].Name {
				t.Errorf("case %d field %d name mismatch. expected: %s, got: %s", i, j, f.Name, ds.Schema.Fields[j].Name)
			}
		}
	}
}

func TestCamelize(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"one two three", "one_two_three"},
		{"users/brendan/stuff/and/such.ext", "such"},
		{"users/brendan/stuff/and/such***.ext", "such"},
		{"users/brendan/stuff/and/separated-by-dashes.ext", "separated_by_dashes"},
		{"CamelCase", "camelcase"},
	}

	for i, c := range cases {
		if c.out == "" {
			c.out = c.in
		}

		got := Camelize(c.in)
		if got != c.out {
			t.Errorf("case %d mismatch got: '%s', expected: '%s'", i, got, c.out)
		}
	}
}
