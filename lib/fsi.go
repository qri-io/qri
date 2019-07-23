package lib

import (
	"github.com/qri-io/qri/fsi"
)

// FSIMethods encapsulates filesystem integrations methods
type FSIMethods struct {
	inst *Instance
}

// NewFSIMethods creates a fsi handle from an instance
func NewFSIMethods(inst *Instance) *FSIMethods {
	return &FSIMethods{inst: inst}
}

// CoreRequestsName specifies this is a fsi handle
func (m FSIMethods) CoreRequestsName() string { return "fsi" }

// FSILink is a file-system-integration link between
type FSILink = fsi.Link

// Links lists all fsi links
func (m *FSIMethods) Links(p *bool, res *[]*FSILink) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.Links", p, res)
	}

	fsint := fsi.NewFSI(m.inst.repo, fsi.RepoPath(m.inst.repoPath))
	*res, err = fsint.Links()
	return err
}

// LinkParams encapsulate parameters to the link method
type LinkParams struct {
	Dir string
	Ref string
}

// CreateLink creates a connection between a working drirectory and a dataset history
func (m *FSIMethods) CreateLink(p *LinkParams, res *string) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.Link", p, res)
	}

	fsint := fsi.NewFSI(m.inst.repo, fsi.RepoPath(m.inst.repoPath))
	*res, err = fsint.CreateLink(p.Dir, p.Ref)
	return err
}

// UpdateLink creates a connection between a working drirectory and a dataset history
func (m *FSIMethods) UpdateLink(p *LinkParams, res *string) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.Link", p, res)
	}

	// TODO (b5) - inst should have an fsi instance
	fsint := fsi.NewFSI(m.inst.repo, fsi.RepoPath(m.inst.repoPath))
	*res, err = fsint.UpdateLink(p.Dir, p.Ref)
	return err
}

// Unlink rmeoves a connection between a working drirectory and a dataset history
func (m *FSIMethods) Unlink(p *LinkParams, res *string) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.Unlink", p, res)
	}

	// TODO (b5) - inst should have an fsi instance
	fsint := fsi.NewFSI(m.inst.repo, fsi.RepoPath(m.inst.repoPath))
	*res, err = fsint.Unlink(p.Dir, p.Ref)
	return err
}

// StatusItem is an alias for an fsi.StatusItem
type StatusItem = fsi.StatusItem

// Status checks for any modifications or errors in a linked directory
func (m *FSIMethods) Status(dir *string, res *[]StatusItem) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.Status", dir, res)
	}

	// TODO (b5) - inst should have an fsi instance
	fsint := fsi.NewFSI(m.inst.repo, fsi.RepoPath(m.inst.repoPath))
	*res, err = fsint.Status(*dir)
	return err
}

// AliasStatus checks for any modifications or errors in a dataset alias
func (m *FSIMethods) AliasStatus(alias *string, res *[]StatusItem) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.AliasStatus", alias, res)
	}

	// TODO (b5) - inst should have an fsi instance
	fsint := fsi.NewFSI(m.inst.repo, fsi.RepoPath(m.inst.repoPath))
	*res, err = fsint.AliasStatus(*alias)
	return err
}

// StoredStatus returns a status-like report of a dataset reference
func (m *FSIMethods) StoredStatus(ref *string, res *[]StatusItem) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.StoredStatus", ref, res)
	}

	// TODO (b5) - inst should have an fsi instance
	fsint := fsi.NewFSI(m.inst.repo, fsi.RepoPath(m.inst.repoPath))
	*res, err = fsint.StoredStatus(*ref)
	return err
}
