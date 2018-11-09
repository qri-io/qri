// Package rev defines structure and syntax for specifying revisions of a
// dataset history. Much of this is inspired by git revisions:
// https://git-scm.com/docs/gitrevisions
//
// Unlike git, Qri is aware of the underlying data model it's selecting against,
// so revisions can have conventional names for specifying fields of a dataset
package rev

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// Rev names a field of a dataset at a snapshot
type Rev struct {
	// field scopt, currently can only be a component name, or the entire dataset
	Field string
	// the nth-generational ancestor of a history
	Gen int
}

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
	field, ok := fieldMap[rev]
	if !ok {
		return nil, errors.New(fmt.Sprintf("unrecognized revision field: %s", rev))
	}
	return &Rev{
		Gen:   1,
		Field: field,
	}, nil
}

var fieldMap = map[string]string{
	"dataset":   "ds",
	"meta":      "md",
	"viz":       "vz",
	"transform": "tf",
	"structure": "st",
	"body":      "bd",

	"ds": "ds",
	"md": "md",
	"vz": "vz",
	"tf": "tf",
	"st": "st",
	"bd": "bd",
}
