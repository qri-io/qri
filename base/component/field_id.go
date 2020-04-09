package component

import (
	"regexp"
	"strings"
)

// DatasetFields is a list of valid dataset field identifiers
var DatasetFields = []string{"commit", "cm", "structure", "st", "body", "bd", "meta", "md", "readme", "rm", "viz", "vz", "transform", "tf", "rendered", "rd", "stats"}

// IsDatasetField can be used to check if a string is a dataset field identifier
var IsDatasetField = regexp.MustCompile("(?i)^(" + strings.Join(DatasetFields, "|") + ")($|\\.)")
