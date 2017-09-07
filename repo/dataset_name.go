package repo

import (
	"path/filepath"
	"regexp"
	"strings"
)

// regex for dataset name validation
var alphaNumericRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// CoerceDatasetName tries to extract a usable variable name from a string of text
// TODO - this will need to be more robust
func CoerceDatasetName(name string) string {
	name = strings.ToLower(name)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.Replace(name, "/.:\\", "", -1)
	name = strings.Replace(name, "-", "_", -1)
	name = strings.Replace(name, " ", "_", -1)
	return name
}

// ValidDatasetName returns true if the given string can be used as the name of a dataset
func ValidDatasetName(name string) bool {
	return alphaNumericRegex.MatchString(name)
}
