package cmd

import (
	"fmt"

	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
)

// RefSelect represents zero or more references
type RefSelect struct {
	refs []string
}

// NewEmptyRefSelect returns an empty reference selection
func NewEmptyRefSelect() *RefSelect {
	return &RefSelect{refs: []string{}}
}

// NewRefSelect returns a single explicitly provided reference
func NewRefSelect(ref string) *RefSelect {
	return &RefSelect{refs: []string{ref}}
}

// NewListOfRefSelects returns a list of explicitly provided references
func NewListOfRefSelects(refs []string) *RefSelect {
	return &RefSelect{refs: refs}
}

// Ref returns the reference as a string
func (r *RefSelect) Ref() string {
	if r == nil || len(r.refs) == 0 {
		return ""
	}
	return r.refs[0]
}

// RefList returns a list of all references
func (r *RefSelect) RefList() []string {
	if r == nil {
		return []string{""}
	}
	return r.refs
}

const (
	// AnyNumberOfReferences is for commands that can work on any number of dataset references
	AnyNumberOfReferences = -1

	// BadUpperCaseOkayWhenSavingExistingDataset is for the save command, which can have bad
	// upper-case characters in its reference but only if it already exists
	BadUpperCaseOkayWhenSavingExistingDataset = -2
)

// GetCurrentRefSelect returns the current reference selection. This could be explicitly provided
// as command-line arguments, or could be determined by being in a linked directory, or could be
// selected by the `use` command. This order is also the precedence, from most important to least.
// This is the recommended method for command-line commands to get references.
// If an Ensurer is passed in, it is used to ensure that the ref in the .qri-ref linkfile matches
// what is in the repo.
func GetCurrentRefSelect(f Factory, args []string, allowed int) (*RefSelect, error) {
	// If reference is specified by the user provide command-line arguments, use that reference.
	if len(args) > 0 {
		// If bad upper-case characters are allowed, skip checking for them
		if allowed == BadUpperCaseOkayWhenSavingExistingDataset {
			// Bad upper-case characters are ignored, references will be checked again inside lib.
			allowed = 1
		} else {
			// For each argument, make sure it's a valid and not using upper-case chracters.
			for _, refstr := range args {
				_, err := dsref.Parse(refstr)
				if err == dsref.ErrBadCaseName {
					// TODO(dustmop): For now, this is just a warning but not a fatal error.
					// In the near future, change to: `return nil, dsref.ErrBadCaseShouldRename`
					// The test `TestBadCaseIsJustWarning` in cmd/cmd_test.go verifies that this
					// is not a fatal error.
					// TODO(dustmop): Change to a fatal error after qri 0.9.9 releases.
					log.Error(dsref.ErrBadCaseShouldRename)
				}
			}
		}
		if allowed == AnyNumberOfReferences {
			return NewListOfRefSelects(args), nil
		}
		if len(args) > allowed {
			return nil, fmt.Errorf("%d references allowed but %d were given", allowed, len(args))
		}
		if allowed == 1 {
			return NewRefSelect(args[0]), nil
		}
		return NewListOfRefSelects(args), nil
	}

	// Empty refselect
	return NewEmptyRefSelect(), repo.ErrEmptyRef
}
