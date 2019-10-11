// Package dsref defines structure and syntax for referring to a dataset
package dsref

import (
	"fmt"
	"strconv"
	"strings"
)

// Rev names a field of a dataset at a snapshot relative to the latest version
// in a history
// Much of this is inspired by git revisions:
// https://git-scm.com/docs/gitrevisions
//
// Unlike git, Qri is aware of the underlying data model it's selecting against,
// so revisions can have conventional names for specifying fields of a dataset
type Rev struct {
	// field scopt, currently can only be a component name, or the entire dataset
	Field string
	// the nth-generational ancestor of a history
	Gen int
}

// AllGenerations represents all the generations of a dataset's history
const AllGenerations = -1

// ParseRevs turns a comma-separated list of revisions into a slice of revisions
func ParseRevs(str string) (revs []*Rev, err error) {
	for _, revStr := range strings.Split(str, ",") {
		rev, err := ParseRev(revStr)
		if err != nil {
			return nil, err
		}
		revs = append(revs, rev)
	}
	return revs, nil
}

// ParseRev turns a string into a revision
func ParseRev(rev string) (*Rev, error) {
	// Check for "all".
	if rev == "all" {
		return &Rev{Gen: AllGenerations, Field: "ds"}, nil
	}
	// Check for integer.
	num, err := strconv.Atoi(rev)
	if err == nil {
		return &Rev{Gen: num, Field: "ds"}, nil
	}
	// Check for field name.
	field, ok := fieldMap[rev]
	if ok {
		return &Rev{Gen: 1, Field: field}, nil
	}
	return nil, fmt.Errorf("unrecognized revision field: %s", rev)
}

// NewAllRevisions returns a Rev struct that represents all revisions.
func NewAllRevisions() Rev {
	return Rev{Field: "ds", Gen: AllGenerations}
}

var fieldMap = map[string]string{
	"dataset":   "ds",
	"meta":      "md",
	"viz":       "vz",
	"transform": "tf",
	"structure": "st",
	"body":      "bd",
	"rendered":  "rd",

	"ds": "ds",
	"md": "md",
	"vz": "vz",
	"tf": "tf",
	"st": "st",
	"bd": "bd",
	"rd": "rd",
}
