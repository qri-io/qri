package lib

import (
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/repo"
)

// SelectionMethods functions support "selecting" dataset(s), storing state about
// which datasets to apply other operations to. Selections power the "qri use"
// CLI command
type SelectionMethods interface {
	Methods
	SetSelectedRefs(sel *[]repo.DatasetRef, done *bool) error
	SelectedRefs(done *bool, sel *[]repo.DatasetRef) error
}

// NewSelectionMethods creates a selectionMethods pointer from either a repo
// or an rpc.Client
func NewSelectionMethods(inst Instance) SelectionMethods {
	if repo := inst.Repo(); repo != nil {
		return selectionMethods{repo: repo}
	}
	if cli := inst.RPC(); cli != nil {
		return selectionMethods{cli: cli}
	}

	panic(fmt.Errorf("can't create selectino handle. Instance has neither a Repo or RPC CLI"))
}

// selectionMethods encapsulates business logic for the qri search
// command
type selectionMethods struct {
	cli  *rpc.Client
	repo repo.Repo
}

// MethodsKind implements the requests
func (r selectionMethods) MethodsKind() string { return "SelectionMethods" }

// SetSelectedRefs sets the current set of selected references
func (r selectionMethods) SetSelectedRefs(sel *[]repo.DatasetRef, done *bool) error {
	if r.cli != nil {
		return r.cli.Call("SelectionMethods.SetSelectedRefs", sel, done)
	}

	if rs, ok := r.repo.(repo.RefSelector); ok {
		return rs.SetSelectedRefs(*sel)
	}
	return repo.ErrRefSelectionNotSupported
}

// SelectedRefs gets the current set of selected references
func (r selectionMethods) SelectedRefs(done *bool, sel *[]repo.DatasetRef) (err error) {
	if r.cli != nil {
		return r.cli.Call("SelectionMethods.SelectedRefs", done, sel)
	}

	if rs, ok := r.repo.(repo.RefSelector); ok {
		*sel, err = rs.SelectedRefs()
		return
	}
	return repo.ErrRefSelectionNotSupported
}

// DefaultSelectedRefs adds selected references to refs if no refs are provided
func DefaultSelectedRefs(r repo.Repo, refs *[]repo.DatasetRef) (err error) {
	if len(*refs) == 0 {
		var done bool
		err = selectionMethods{repo: r}.SelectedRefs(&done, refs)
		if err == repo.ErrRefSelectionNotSupported {
			return nil
		}
	}
	return
}

// DefaultSelectedRef sets ref to the first selected reference if the provided ref is empty
func DefaultSelectedRef(r repo.Repo, ref *repo.DatasetRef) (err error) {
	if ref == nil || ref.IsEmpty() {
		var (
			done bool
			refs = []repo.DatasetRef{}
		)

		err = selectionMethods{repo: r}.SelectedRefs(&done, &refs)
		if err != nil {
			if err == repo.ErrRefSelectionNotSupported {
				return nil
			}
			return
		}

		if len(refs) > 0 {
			*ref = refs[0]
		}
	}
	return
}
