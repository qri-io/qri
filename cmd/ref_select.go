package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/qri-io/qri/fsi"
)

// RefSelect represents zero or more references, either explicitly provided or implied
type RefSelect struct {
	kind string
	ref  string
	dir  string
}

// NewExplicitRefSelect returns a single explicitly provided reference
func NewExplicitRefSelect(ref string) *RefSelect {
	return &RefSelect{ref: ref}
}

// NewLinkedDirectoryRefSelect returns a single reference implied by a linked directory
func NewLinkedDirectoryRefSelect(ref, dir string) *RefSelect {
	pos := strings.Index(ref, "@")
	if pos != -1 {
		ref = ref[:pos]
	}
	return &RefSelect{kind: "for", ref: ref, dir: dir}
}

// NewUsingRefSelect returns a single reference implied by the use command
func NewUsingRefSelect(ref string) *RefSelect {
	return &RefSelect{kind: "using", ref: ref}
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
	if r == nil {
		return ""
	}
	return r.ref
}

// RefList returns a list of all references
func (r *RefSelect) RefList() []string {
	if r == nil {
		return []string{""}
	}
	return []string{r.ref}
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
	return fmt.Sprintf("%s dataset [%s]", r.kind, r.ref)
}

// GetCurrentRefSelect returns the current reference selection
func GetCurrentRefSelect(f Factory, args []string, allowed int) (*RefSelect, error) {
	// TODO(dlong): Respect `allowed`, number of refs the command uses. -1 means any.
	// TODO(dlong): For example, `get` allows -1, `diff` allows 2, `save` allows 1
	// If reference is specified by the user provide command-line arguments, use that reference.
	if len(args) > 0 {
		return NewExplicitRefSelect(args[0]), nil
	}
	// If in a working directory that is linked to a dataset, use that link's reference.
	dir, err := os.Getwd()
	if err == nil {
		ref, ok := fsi.GetLinkedFilesysRef(dir)
		if ok {
			return NewLinkedDirectoryRefSelect(ref, dir), nil
		}
	}
	// Find what `use` is referencing and use that.
	refs, err := DefaultSelectedRefList(f)
	if err != nil {
		return nil, err
	}
	if len(refs) == 1 {
		return NewUsingRefSelect(refs[0]), nil
	}
	// Empty refset
	return nil, fmt.Errorf("empty dataset reference")
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
	// Empty refset
	return nil, fmt.Errorf("empty dataset reference")
}
