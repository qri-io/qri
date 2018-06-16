package lib

import (
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/repo"
)

// SelectionRequests encapsulates business logic for the qri search
// command
type SelectionRequests struct {
	cli  *rpc.Client
	repo repo.Repo
}

// NewSelectionRequests creates a SelectionRequests pointer from either a repo
// or an rpc.Client
func NewSelectionRequests(r repo.Repo, cli *rpc.Client) *SelectionRequests {
	if r != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewSelectionRequests"))
	}
	return &SelectionRequests{
		cli:  cli,
		repo: r,
	}
}

// CoreRequestsName implements the requests
func (r SelectionRequests) CoreRequestsName() string { return "selection" }

// SetSelectedRefs sets the current set of selected references
func (r *SelectionRequests) SetSelectedRefs(sel *[]repo.DatasetRef, done *bool) error {
	if r.cli != nil {
		return r.cli.Call("SelectionRequests.SetSelectedRefs", sel, done)
	}

	if rs, ok := r.repo.(repo.RefSelector); ok {
		return rs.SetSelectedRefs(*sel)
	}
	return fmt.Errorf("selection not supported")
}

// SelectedRefs gets the current set of selected references
func (r *SelectionRequests) SelectedRefs(done *bool, sel *[]repo.DatasetRef) (err error) {
	if r.cli != nil {
		return r.cli.Call("SelectionRequests.SelectedRefs", done, sel)
	}

	if rs, ok := r.repo.(repo.RefSelector); ok {
		*sel, err = rs.SelectedRefs()
		return
	}
	return fmt.Errorf("selection not supported")
}
