package component

import (
	"regexp"
)

// IsDatasetField can be used to check if a string is a dataset field identifier
var IsDatasetField = regexp.MustCompile("(?i)^(commit|cm|structure|st|body|bd|meta|md|readme|rm|viz|vz|transform|tf|rendered|rd)($|\\.)")
