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

// LinkedRefs lists all fsi links
func (m *FSIMethods) LinkedRefs(p *ListParams, res *[]repo.DatasetRef) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.LinkedRefs", p, res)
	}

	fsint := fsi.NewFSI(m.inst.repo)
	*res, err = fsint.LinkedRefs(p.Offset, p.Limit)
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

	fsint := fsi.NewFSI(m.inst.repo)
	*res, err = fsint.CreateLink(p.Dir, p.Ref)
	return err
}

// UpdateLink creates a connection between a working drirectory and a dataset history
func (m *FSIMethods) UpdateLink(p *LinkParams, res *string) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.UpdateLink", p, res)
	}

	// TODO (b5) - inst should have an fsi instance
	fsint := fsi.NewFSI(m.inst.repo)
	*res, err = fsint.UpdateLink(p.Dir, p.Ref)
	return err
}

// Unlink rmeoves a connection between a working drirectory and a dataset history
func (m *FSIMethods) Unlink(p *LinkParams, res *string) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.Unlink", p, res)
	}

	// TODO (b5) - inst should have an fsi instance
	fsint := fsi.NewFSI(m.inst.repo)
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
	fsint := fsi.NewFSI(m.inst.repo)
	*res, err = fsint.Status(*dir)
	return err
}

// AliasStatus checks for any modifications or errors in a dataset alias
func (m *FSIMethods) AliasStatus(alias *string, res *[]StatusItem) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.AliasStatus", alias, res)
	}

	// TODO (b5) - inst should have an fsi instance
	fsint := fsi.NewFSI(m.inst.repo)
	*res, err = fsint.AliasStatus(*alias)
	return err
}

// StoredStatus returns a status-like report of a dataset reference
func (m *FSIMethods) StoredStatus(ref *string, res *[]StatusItem) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.StoredStatus", ref, res)
	}

	// TODO (b5) - inst should have an fsi instance
	fsint := fsi.NewFSI(m.inst.repo)
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

	fsint := fsi.NewFSI(m.inst.repo)

	// Create a directory.
	if err := os.Mkdir(p.Dir, os.ModePerm); err != nil {
		return err
	}

	// Create the link file, containing the dataset reference.
	if _, err = fsint.CreateLink(p.Dir, p.Ref); err != nil {
		return err
	}

	// Write components of the dataset to the dataset.
	err = fsi.WriteComponents(ds, p.Dir)
	return err
}

// FSIDatasetForRef reads an fsi-linked dataset for a given reference string
func (m *FSIMethods) FSIDatasetForRef(refStr *string, res *repo.DatasetRef) error {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.FSIDatasetForRef", refStr, res)
	}

	ref, err := repo.ParseDatasetRef(*refStr)
	if err != nil {
		return err
	}

	if err = repo.CanonicalizeDatasetRef(m.inst.repo, &ref); err != nil {
		return err
	}

	ds, _, _, err := fsi.ReadDir(ref.FSIPath)
	if err != nil {
		return err
	}

	// TODO (b5) - these transient fields should probably be set by fsi.ReadDir
	ds.Peername = ref.Peername
	ds.Name = ref.Name
	ds.Path = ref.FSIPath
	ds.PreviousPath = ref.Path
	ref.Dataset = ds

	*res = ref
	return nil
}

// FSIBodyParams defines parameters for looking up the body of a dataset
// This structure is based on GetParams.
// TODO (@b5) - refactor this away. It's too much like other things
type FSIBodyParams struct {
	// Path to get, this will often be a dataset reference like me/dataset
	Path string

	Format       string
	FormatConfig dataset.FormatConfig

	Offset, Limit int
	All           bool
}

// FSIDatasetBody grabs the body of a dataset
func (m *FSIMethods) FSIDatasetBody(p *FSIBodyParams, res *[]byte) error {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.FSIDatasetBody", p, res)
	}

	df, err := dataset.ParseDataFormatString(p.Format)
	if err != nil {
		return err
	}

	ref, err := repo.ParseDatasetRef(p.Path)
	if err != nil {
		return err
	}

	if err = repo.CanonicalizeDatasetRef(m.inst.repo, &ref); err != nil {
		return err
	}

	*res, err = fsi.GetBody(ref.FSIPath, df, p.FormatConfig, p.Offset, p.Limit, p.All)
	return err
}

// InitFSIDatasetParams proxies parameters to initialization
type InitFSIDatasetParams = fsi.InitParams

// InitDataset creates a new dataset and FSI link
func (m *FSIMethods) InitDataset(p *InitFSIDatasetParams, name *string) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.InitDataset", p, name)
	}

	// TODO (b5) - inst should have an fsi instance
	fsint := fsi.NewFSI(m.inst.repo)
	*name, err = fsint.InitDataset(*p)
	return err
}
