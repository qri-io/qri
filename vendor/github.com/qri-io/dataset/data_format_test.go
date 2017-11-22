package dataset

import (
	"bytes"
	"testing"
)

func TestDataFormatString(t *testing.T) {
	cases := []struct {
		f      DataFormat
		expect string
	}{
		{UnknownDataFormat, ""},
		{CsvDataFormat, "csv"},
		{JsonDataFormat, "json"},
		{XmlDataFormat, "xml"},
		{XlsDataFormat, "xls"},
		{CdxjDataFormat, "cdxj"},
	}

	for i, c := range cases {
		if got := c.f.String(); got != c.expect {
			t.Errorf("case %d mismatch. expected: %s, got: %s", i, c.expect, got)
			continue
		}
	}
}

func TestParseDataFormatString(t *testing.T) {
	cases := []struct {
		in     string
		expect DataFormat
		err    string
	}{
		{"", UnknownDataFormat, ""},
		{".csv", CsvDataFormat, ""},
		{"csv", CsvDataFormat, ""},
		{".json", JsonDataFormat, ""},
		{"json", JsonDataFormat, ""},
		{".xml", XmlDataFormat, ""},
		{"xml", XmlDataFormat, ""},
		{".xls", XlsDataFormat, ""},
		{"xls", XlsDataFormat, ""},
		{".cdxj", CdxjDataFormat, ""},
		{"cdxj", CdxjDataFormat, ""},
	}

	for i, c := range cases {
		got, err := ParseDataFormatString(c.in)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch '%s' != '%s'", i, c.expect, err)
			continue
		}
		if got != c.expect {
			t.Errorf("case %d response mismatch. expected: %s got: %s", i, c.expect, got)
			continue
		}
	}
}

func TestDataFormatMarshalJSON(t *testing.T) {
	cases := []struct {
		format DataFormat
		expect []byte
		err    string
	}{
		{CsvDataFormat, []byte(`"csv"`), ""},
		{JsonDataFormat, []byte(`"json"`), ""},
		{XmlDataFormat, []byte(`"xml"`), ""},
		{XlsDataFormat, []byte(`"xls"`), ""},
		{CdxjDataFormat, []byte(`"cdxj"`), ""},
	}
	for i, c := range cases {
		got, err := c.format.MarshalJSON()
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}
		if !bytes.Equal(got, c.expect) {
			t.Errorf(`case %d response mismatch. expected: %s, got: %s`, i, string(c.expect), string(got))
			continue
		}
	}
}

func TestDataFormatUnmarshalJSON(t *testing.T) {
	cases := []struct {
		data   []byte
		expect DataFormat
		err    string
	}{
		{[]byte(`"csv"`), CsvDataFormat, ""},
		{[]byte(`"json"`), JsonDataFormat, ""},
		{[]byte(`"xml"`), XmlDataFormat, ""},
		{[]byte(`"xls"`), XlsDataFormat, ""},
		{[]byte(`"cdxj"`), CdxjDataFormat, ""},
	}

	for i, c := range cases {
		a := DataFormat(0)
		got := &a
		err := got.UnmarshalJSON(c.data)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}
		if *got != c.expect {
			t.Errorf(`case %d response mismatch. expected: %s, got: %s`, i, c.expect, *got)
			continue
		}

	}
}
