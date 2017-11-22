package validate

import (
	"fmt"

	"github.com/qri-io/dataset"
)

// Structure checks that a dataset structure is valid for use
// returning the first error encountered, nil if the Structure is valid
func Structure(s *dataset.Structure) error {
	checkedFieldNames := map[string]bool{}
	fields := s.Schema.Fields
	for _, field := range fields {
		if alphaNumericRegex.FindString(field.Name) == "" {
			return fmt.Errorf("error: illegal name '%s', must start with a letter and consist of only alpha-numeric characters and/or underscores and have a total length of no more than 144 characters", field.Name)
		}
		seen := checkedFieldNames[field.Name]
		if seen {
			return fmt.Errorf("error: cannot use the same name, '%s' more than once", field.Name)
		}
		checkedFieldNames[field.Name] = true
	}
	return nil
}
