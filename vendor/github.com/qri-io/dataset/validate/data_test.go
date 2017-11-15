package validate

import (
	"github.com/qri-io/dataset"

	"strings"
	"testing"
)

//Note text examples in testdata.go

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
		{dataset.JsonArrayDataFormat,
			rawText1,
			"error: data format 'JsonArrayData' not currently supported",
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

// takes a slice of strings and createws a pointer to a Structure
// containing a schema containing those fields
func structureTestHelper(s []string) *dataset.Structure {
	fields := []*dataset.Field{}
	for _, fieldName := range s {
		newField := dataset.Field{Name: fieldName}
		fields = append(fields, &newField)
	}
	schema := &dataset.Schema{Fields: fields}
	structure := &dataset.Structure{Schema: schema}
	return structure
}

func TestCheckStructure(t *testing.T) {
	cases := []struct {
		input []string
		err   string
	}{
		{[]string{"abc", "12startsWithNumber"}, `error: illegal name '12startsWithNumber', must start with a letter and consist of only alpha-numeric characters and/or underscores and have a total length of no more than 144 characters`},
		{[]string{"abc", "$dollarsAtBeginning"}, `error: illegal name '$dollarsAtBeginning', must start with a letter and consist of only alpha-numeric characters and/or underscores and have a total length of no more than 144 characters`},
		{[]string{"abc", "Dollars$inTheMiddle"}, `error: illegal name 'Dollars$inTheMiddle', must start with a letter and consist of only alpha-numeric characters and/or underscores and have a total length of no more than 144 characters`},
		{[]string{"abc", ""}, `error: illegal name '', must start with a letter and consist of only alpha-numeric characters and/or underscores and have a total length of no more than 144 characters`},
		{[]string{"abc", "No|pipes"}, `error: illegal name 'No|pipes', must start with a letter and consist of only alpha-numeric characters and/or underscores and have a total length of no more than 144 characters`},
		{[]string{"repeatedName", "repeatedName", "repeatedName"}, "error: cannot use the same name, 'repeatedName' more than once"},
	}
	for i, c := range cases {
		s := structureTestHelper(c.input)
		err := CheckStructure(s)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case [%d] error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}
	}
}
