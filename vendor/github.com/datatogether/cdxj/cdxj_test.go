package cdxj

import (
	"fmt"
	"github.com/datatogether/warc"
	"testing"
	"time"
)

func TestSURTUrl(t *testing.T) {
	cases := []struct {
		in, out string
		err     error
	}{
		{"cnn.com/world", "(com,cnn,)/world>", nil},
		{"http://cnn.com/world", "(com,cnn,)/world>", nil},
		{"https://cnn.com/world", "(com,cnn,)/world>", nil},
		{"ftp://cnn.co.uk/world?foo=bar", "(uk,co,cnn,)/world?foo=bar>", nil},
	}

	for i, c := range cases {
		got, err := SURTUrl(c.in)
		if err != c.err {
			t.Errorf("case %d error mismatch: %s != %s", i, c.err, err)
			continue
		}

		if c.out != got {
			t.Errorf("case %d mismatch. expected: '%s', got: '%s'", i, c.out, got)
			continue
		}
	}
}

func TestUnSURTUrl(t *testing.T) {
	cases := []struct {
		in, out string
		err     error
	}{
		{"(com,cnn,)/world", "cnn.com/world", nil},
		{"com,cnn,)/world>", "cnn.com/world", nil},
		{"com,cnn)/world", "cnn.com/world", nil},
		{"(uk,co,cnn,)/world?foo=bar", "cnn.co.uk/world?foo=bar", nil},
	}

	for i, c := range cases {
		got, err := UnSURTUrl(c.in)
		if err != c.err {
			t.Errorf("case %d error mismatch: %s != %s", i, c.err, err)
			continue
		}

		if c.out != got {
			t.Errorf("case %d mismatch. expected: '%s', got: '%s'", i, c.out, got)
			continue
		}
	}
}

func TestUnSURTPath(t *testing.T) {
	cases := []struct {
		in, out string
		err     error
	}{
		{"(com,cnn,)/world", "/world", nil},
		{"com,cnn,)/world>", "/world", nil},
		{"com,cnn)/world", "/world", nil},
		{"com,cnn)", "/", nil},
		{"(uk,co,cnn,)/world?foo=bar", "/world?foo=bar", nil},
	}

	for i, c := range cases {
		got, err := UnSURTPath(c.in)
		if err != c.err {
			t.Errorf("case %d error mismatch: %s != %s", i, c.err, err)
			continue
		}

		if c.out != got {
			t.Errorf("case %d mismatch. expected: '%s', got: '%s'", i, c.out, got)
			continue
		}
	}
}

func TestRecordUnmarshalCDXJ(t *testing.T) {
	cases := []struct {
		data []byte
		out  *Record
		err  error
	}{
		{[]byte(`(com,cnn,)/world 2015-09-03T13:27:52Z response {"a" : 0, "b" : "b", "c" : false }`), &Record{"cnn.com/world", time.Date(2015, time.September, 3, 13, 27, 52, 0, time.UTC), warc.RecordTypeResponse, map[string]interface{}{"a": 0, "b": "b", "c": false}}, nil},
	}

	for i, c := range cases {
		r := &Record{}
		if err := r.UnmarshalCDXJ(c.data); err != c.err {
			t.Errorf("case %d error mismatch: %s != %s", i, c.err, err)
			continue
		}

		if err := CompareRecords(c.out, r); err != nil {
			t.Errorf("case %d record mismatch: %s", i, err.Error())
			continue
		}
	}
}

func CompareRecordSlices(a, b []*Record) error {
	if len(a) != len(b) {
		return fmt.Errorf("record slice length mismatch: %d != %d", len(a), len(b))
	}

	for i, ar := range a {
		br := b[i]
		if err := CompareRecords(ar, br); err != nil {
			return fmt.Errorf("record %d mismatch: %s", i, err.Error())
		}
	}

	return nil
}

func CompareRecords(a, b *Record) error {
	if a == nil && b != nil || b == nil && a != nil {
		return fmt.Errorf("nil mistmatch: %s,%s", a, b)
	} else if a == nil && b == nil {
		return nil
	}

	if a.Uri != b.Uri {
		return fmt.Errorf("record uri mismatch: %s != %s", a.Uri, b.Uri)
	}

	if !a.Timestamp.Equal(b.Timestamp) {
		return fmt.Errorf("timestamp mismatch: %s != %s", a.Timestamp.String(), a.Timestamp.String())
	}

	if a.RecordType != b.RecordType {
		return fmt.Errorf("record type mismatch: %s != %s", a.RecordType, b.RecordType)
	}

	// TODO - compare json field

	return nil
}
