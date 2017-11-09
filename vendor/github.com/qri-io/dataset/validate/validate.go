package validate

import (
	"encoding/json"
	"fmt"
)

// VariableName is a string that conforms to standard variable naming conventions
// must start with a letter, no spaces
// TODO - we're not really using this much, consider depricating, or using properly
type VariableName string

// MarshalJSON satisfies the json.Marshaller interface
func (name VariableName) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, name)), nil
}

// UnmarshalJSON satisfies the json.Unmarshaller interface
func (name *VariableName) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("type should be a string, got %s", data)
	}

	if alphaNumericRegex.MatchString(s) {
		return fmt.Errorf("variable name must contain only letters, numbers, '_' or '-', and start with a letter")
	}

	*name = VariableName(s)
	return nil
}
