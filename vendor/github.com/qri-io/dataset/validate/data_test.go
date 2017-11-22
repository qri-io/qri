package validate

import (
	"github.com/qri-io/dataset/dsio"
	"strings"
	"testing"

	"github.com/qri-io/dataset"
)

func TestDataFormat(t *testing.T) {
	cases := []struct {
		df    dataset.DataFormat
		input string
		err   string
	}{
		{dataset.JsonDataFormat,
			rawText1,
			"error: data format 'JsonData' not currently supported",
		},
		{
			dataset.XlsDataFormat,
			rawText1,
			"error: data format 'XlsData' not currently supported",
		},
		{
			dataset.XmlDataFormat,
			rawText1,
			"error: data format 'XmlData' not currently supported",
		},
		{
			dataset.UnknownDataFormat,
			rawText1,
			"error: unknown data format not currently supported",
		},
		{
			dataset.DataFormat(999),
			rawText1,
			"error: data format not currently supported",
		},
		{
			dataset.CsvDataFormat,
			rawText4,
			"error: inconsistent column length on line 4 of length 2 (rather than 1). ensure all csv columns same length",
		},
		{
			dataset.CsvDataFormat,
			rawText1,
			"",
		},
		{dataset.CdxjDataFormat, emptyRawText, "invalid format, missing cdxj header"},
		{dataset.CdxjDataFormat, cdxjRawText, ""},
	}
	for i, c := range cases {
		r := strings.NewReader(c.input)
		err := DataFormat(c.df, r)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case [%d] error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}
	}
}

func TestDataErrors(t *testing.T) {
	cases := []struct {
		structure *dataset.Structure
		data      string
		cfg       *DataErrorsCfg
		// TODO - validate output structure
		count int
		err   string
	}{
		// {namesStructure, rawText2, DefaultDataErrorsCfg(), 0, ""},
		{namesStructure, rawText2c, DefaultDataErrorsCfg(), 1, ""},
	}

	for i, c := range cases {
		r := dsio.NewRowReader(c.structure, strings.NewReader(c.data))

		got, count, err := DataErrors(r, func(cfg *DataErrorsCfg) { *cfg = *c.cfg })
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case [%d] error mismatch. expected '%s', got: '%s'", i, c.err, err)
			continue
		}

		if c.count != count {
			t.Errorf("case [%d] count mismatch. expected: %d, got: %d", i, c.count, count)
			continue
		}

		if len(c.structure.Schema.Fields) != len(got.Structure().Schema.Fields)-1 {
			t.Errorf("case [%d] structure field length mismatch. expected: %d, got: %d", i, len(c.structure.Schema.Fields)+1, len(got.Structure().Schema.Fields))
			continue
		}
	}
}
