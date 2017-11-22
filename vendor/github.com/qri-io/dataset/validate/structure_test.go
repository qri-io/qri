package validate

import (
	"testing"

	"github.com/qri-io/dataset"
)

func TestStructure(t *testing.T) {
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
		err := Structure(s)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case [%d] error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}
	}
}

// takes a slice of strings and creates a pointer to a Structure
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
