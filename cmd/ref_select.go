package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
)

// RefSelect represents zero or more references, either explicitly provided or implied
type RefSelect struct {
	kind string
	refs []string
}

// when a user is running a command within a working directory that is linked to a
// dataset, the text sent to standard output shall begin with:
//
// for linked dataset [peername/dataset_name]
//
// That is, "using" is to `use`, as "for linked" is to `qri-ref`. In all other cases (an
// explicit dataset ref provided on the command line) neither of these phrases should be
// displayed. This way, the user can tell at a glance what dataset is being used, and the reason
// for why is was selected. The `kind` field on RefSelect controls which of these kinds of
// references is being used.

// NewEmptyRefSelect returns an empty reference selection
func NewEmptyRefSelect() *RefSelect {
	return &RefSelect{refs: []string{}}
}

// NewExplicitRefSelect returns a single explicitly provided reference
func NewExplicitRefSelect(ref string) *RefSelect {
	return &RefSelect{refs: []string{ref}}
}

// NewListOfRefSelects returns a list of explicitly provided references
func NewListOfRefSelects(refs []string) *RefSelect {
	return &RefSelect{refs: refs}
}

// NewUsingRefSelect returns a single reference implied by the use command
func NewUsingRefSelect(ref string) *RefSelect {
	return &RefSelect{kind: "using", refs: []string{ref}}
}

// IsExplicit returns whether the reference is explicit
func (r *RefSelect) IsExplicit() bool {
	return r.kind == ""
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

// String returns a stringified version of the ref selection
func (r *RefSelect) String() string {
	if r.IsExplicit() {
		return ""
	}
	return fmt.Sprintf("%s dataset [%s]", r.kind, strings.Join(r.refs, ", "))
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
			return NewExplicitRefSelect(args[0]), nil
		}
		return NewListOfRefSelects(args), nil
	}

	// Find what `use` is referencing and use that.
	selected, err := DefaultSelectedRefList(f)
	if err != nil {
		return nil, err
	}
	if len(selected) == 1 {
		return NewUsingRefSelect(selected[0]), nil
	}
	// Empty refselect
	return NewEmptyRefSelect(), repo.ErrEmptyRef
}

// DefaultSelectedRefList returns the list of currently `use`ing dataset references
func DefaultSelectedRefList(f Factory) ([]string, error) {
	fileSelectionPath := filepath.Join(f.RepoPath(), FileSelectedRefs)

	refs, err := readFile(fileSelectionPath)
	if err != nil {
		// If selected_refs.json is empty or doesn't exist, not an error.
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	res := make([]string, 0, len(refs))
	for _, r := range refs {
		res = append(res, r.String())
	}

	return res, nil
}
