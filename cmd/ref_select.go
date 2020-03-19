package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

// RefSelect represents zero or more references, either explicitly provided or implied
type RefSelect struct {
	kind string
	refs []string
	dir  string
}

// A notice on UI language used here. When a user has run the `qri use` command to select a
// dataset ref to run commands on, the text sent to standard output should begin with:
//
// using dataset [peername/dataset_name]
//
// In contrast, when a user is running a command within a working directory that is linked to a
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
func NewExplicitRefSelect(ref string, fsi *lib.FSIMethods) *RefSelect {
	if fsi != nil {
		dsRef := reporef.DatasetRef{}
		err := fsi.FSIDatasetForRef(&ref, &dsRef)
		if err == nil {
			return &RefSelect{kind: "for linked", refs: []string{ref}, dir: dsRef.Path}	
		}
	}
	return &RefSelect{refs: []string{ref}}
}

// NewListOfRefSelects returns a list of explicitly provided references
func NewListOfRefSelects(refs []string) *RefSelect {
	return &RefSelect{refs: refs}
}

// NewLinkedDirectoryRefSelect returns a single reference implied by a linked directory
func NewLinkedDirectoryRefSelect(ref, dir string) *RefSelect {
	// Remove the path from the reference, want just peername/ds_name
	pos := strings.Index(ref, "@")
	if pos != -1 {
		ref = ref[:pos]
	}
	return &RefSelect{kind: "for linked", refs: []string{ref}, dir: dir}
}

// NewUsingRefSelect returns a single reference implied by the use command
func NewUsingRefSelect(ref string) *RefSelect {
	return &RefSelect{kind: "using", refs: []string{ref}}
}

// IsExplicit returns whether the reference is explicit
func (r *RefSelect) IsExplicit() bool {
	return r.kind == ""
}

// IsLinked returns whether the reference is implied by a linked directory
func (r *RefSelect) IsLinked() bool {
	return r.dir != ""
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

// Dir returns the directory of a linked directory reference
func (r *RefSelect) Dir() string {
	return r.dir
}

// String returns a stringified version of the ref selection
func (r *RefSelect) String() string {
	if r.IsExplicit() {
		return ""
	}
	return fmt.Sprintf("%s dataset [%s]", r.kind, strings.Join(r.refs, ", "))
}

// GetCurrentRefSelect returns the current reference selection. This could be explicitly provided
// as command-line arguments, or could be determined by being in a linked directory, or could be
// selected by the `use` command. This order is also the precendence, from most important to least.
// This is the recommended method for command-line commands to get references, unless they have a
// special way of interacting with datasets (for example, `qri status`).
// If an fsi pointer is passed in, use it to ensure that the ref in the .qri-ref linkfile matches
// what is in the repo.
func GetCurrentRefSelect(f Factory, args []string, allowed int, fsi *lib.FSIMethods) (*RefSelect, error) {
	// TODO(dlong): Respect `allowed`, number of refs the command uses. -1 means any.
	// TODO(dlong): For example, `get` allows -1, `diff` allows 2, `save` allows 1
	// If reference is specified by the user provide command-line arguments, use that reference.
	if len(args) > 0 {
		if allowed >= 2 {
			// Diff allows multiple explicit references.
			return NewListOfRefSelects(args), nil
		}
		return NewExplicitRefSelect(args[0], fsi), nil
	}
	// If in a working directory that is linked to a dataset, use that link's reference.
	refs, err := GetLinkedRefSelect()
	if err == nil {
		if fsi != nil {
			// Ensure that the link in the working directory matches what is in the repo.
			var out bool
			err = fsi.EnsureRef(&lib.EnsureParams{Dir: refs.Dir(), Ref: refs.Ref()}, &out)
			if err != nil {
				log.Debugf("%s", err)
			}
		}
		return refs, nil
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

// GetLinkedRefSelect returns the current reference selection only if it is a linked directory
func GetLinkedRefSelect() (*RefSelect, error) {
	// If in a working directory that is linked to a dataset, use that link's reference.
	dir, err := os.Getwd()
	if err == nil {
		ref, ok := fsi.GetLinkedFilesysRef(dir)
		if ok {
			return NewLinkedDirectoryRefSelect(ref, dir), nil
		}
	}
	// Empty refselect
	return nil, repo.ErrEmptyRef
}

// DefaultSelectedRefList returns the list of currently `use`ing dataset references
func DefaultSelectedRefList(f Factory) ([]string, error) {
	fileSelectionPath := filepath.Join(f.QriRepoPath(), FileSelectedRefs)

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
