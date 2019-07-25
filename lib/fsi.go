package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/repo"
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
	// absolutize path name
	path, err := filepath.Abs(p.Dir)
	if err != nil {
		return err
	}

	p.Dir = path

	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.CreateLink", p, res)
	}

	fsint := fsi.NewFSI(m.inst.repo, fsi.RepoPath(m.inst.repoPath))
	*res, err = fsint.CreateLink(p.Dir, p.Ref)
	return err
}

// UpdateLink creates a connection between a working drirectory and a dataset history
func (m *FSIMethods) UpdateLink(p *LinkParams, res *string) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.UpdateLink", p, res)
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
	err = fsint.Unlink(p.Dir, p.Ref)

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

// CheckoutParams provides parameters to the Checkout method.
type CheckoutParams struct {
	Dir string
	Ref string
}

// Checkout method writes a dataset to a directory as individual files.
func (m *FSIMethods) Checkout(p *CheckoutParams, out *string) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.Checkout", p, out)
	}

	// If directory exists, error.
	if _, err = os.Stat(p.Dir); !os.IsNotExist(err) {
		return fmt.Errorf("directory with name \"%s\" already exists", p.Dir)
	}

	// Handle the ref to checkout.
	ref := &repo.DatasetRef{}
	if p.Ref == "" {
		return repo.ErrEmptyRef
	}
	*ref, err = repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return fmt.Errorf("'%s' is not a valid dataset reference", p.Ref)
	}
	if err = repo.CanonicalizeDatasetRef(m.inst.repo, ref); err != nil {
		return
	}

	// Load dataset that is being checked out.
	ds, err := dsfs.LoadDataset(m.inst.repo.Store(), ref.Path)
	if err != nil {
		return fmt.Errorf("error loading dataset")
	}
	ds.Name = ref.Name
	ds.Peername = ref.Peername
	if err = base.OpenDataset(m.inst.repo.Filesystem(), ds); err != nil {
		return
	}

	fsint := fsi.NewFSI(m.inst.repo, fsi.RepoPath(m.inst.repoPath))

	// Create a directory.
	if err := os.Mkdir(p.Dir, os.ModePerm); err != nil {
		return err
	}

	// Create the link file, containing the dataset reference.
	if _, err = fsint.CreateLink(p.Dir, p.Ref); err != nil {
		return err
	}

	// Write components of the dataset to the dataset.
	err = fsint.WriteComponents(ds, p.Dir)
	return err
}

// FSIDatasetForRef reads an fsi-linked dataset for
func (m *FSIMethods) FSIDatasetForRef(refStr *string, res *dataset.Dataset) error {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.FSIDatasetForRef", refStr, res)
	}

	// TODO (b5) - inst should have an fsi instance
	fsint := fsi.NewFSI(m.inst.repo, fsi.RepoPath(m.inst.repoPath))

	link, err := fsint.RefLink(*refStr)
	if err != nil {
		return err
	}

	ds, _, _, err := fsi.ReadDir(link.Path)
	if err != nil {
		return err
	}

	*res = *ds
	return nil
}
