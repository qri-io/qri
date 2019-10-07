package lib

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/actions"
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

	*res, err = m.inst.fsi.LinkedRefs(p.Offset, p.Limit)
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

	*res, err = m.inst.fsi.CreateLink(p.Dir, p.Ref)
	return err
}

// UpdateLink creates a connection between a working drirectory and a dataset history
func (m *FSIMethods) UpdateLink(p *LinkParams, res *string) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.UpdateLink", p, res)
	}

	*res, err = m.inst.fsi.UpdateLink(p.Dir, p.Ref)
	return err
}

// Unlink rmeoves a connection between a working drirectory and a dataset history
func (m *FSIMethods) Unlink(p *LinkParams, res *string) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.Unlink", p, res)
	}

	return m.inst.fsi.Unlink(p.Dir, p.Ref)
}

// StatusItem is an alias for an fsi.StatusItem
type StatusItem = fsi.StatusItem

// Status checks for any modifications or errors in a linked directory against its previous
// version in the repo. Must only be called if FSI is enabled for this dataset.
func (m *FSIMethods) Status(dir *string, res *[]StatusItem) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.Status", dir, res)
	}
	ctx := context.TODO()

	*res, err = m.inst.fsi.Status(ctx, *dir)
	return err
}

// StatusForAlias receives an alias for a dataset that must be linked to the filesystem, and checks
// the status of its current working directory. It is an error to call this for a reference that
// is not linked.
func (m *FSIMethods) StatusForAlias(alias *string, res *[]StatusItem) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.AliasStatus", alias, res)
	}
	ctx := context.TODO()

	dir, err := m.inst.fsi.AliasToLinkedDir(*alias)
	if err != nil {
		return err
	}
	*res, err = m.inst.fsi.Status(ctx, dir)
	return err
}

// StatusAtVersion gets changes that happened at a particular version in the history of the given
// dataset reference. Not used for FSI.
func (m *FSIMethods) StatusAtVersion(ref *string, res *[]StatusItem) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.StoredStatus", ref, res)
	}
	ctx := context.TODO()

	*res, err = m.inst.fsi.StatusAtVersion(ctx, *ref)
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
	ctx := context.TODO()

	// TODO(dlong): Fail if Dir is "", should be required to specify a location. Should probably
	// only allow absolute paths. Add tests.

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
	ds, err := dsfs.LoadDataset(ctx, m.inst.repo.Store(), ref.Path)
	if err != nil {
		return fmt.Errorf("error loading dataset")
	}
	ds.Name = ref.Name
	ds.Peername = ref.Peername
	if err = base.OpenDataset(ctx, m.inst.repo.Filesystem(), ds); err != nil {
		return
	}

	// Create a directory.
	if err := os.Mkdir(p.Dir, os.ModePerm); err != nil {
		return err
	}

	// Create the link file, containing the dataset reference.
	if _, err = m.inst.fsi.CreateLink(p.Dir, p.Ref); err != nil {
		return err
	}

	// Write components of the dataset to the working directory.
	err = fsi.WriteComponents(ds, p.Dir)
	return err
}

// FSIWriteParams encapsultes arguments for writing to an FSI-linked directory
type FSIWriteParams struct {
	Ref string
	Ds  *dataset.Dataset
}

// Write mutates a linked dataset on the filesystem
func (m *FSIMethods) Write(p *FSIWriteParams, res *[]StatusItem) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.Write", p, res)
	}
	ctx := context.TODO()

	if p.Ref == "" {
		return repo.ErrEmptyRef
	}
	if p.Ds == nil {
		return fmt.Errorf("dataset is required")
	}
	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return fmt.Errorf("'%s' is not a valid dataset reference", p.Ref)
	}
	err = repo.CanonicalizeDatasetRef(m.inst.node.Repo, &ref)
	if err != nil && err != repo.ErrNoHistory {
		return err
	}

	// Directory to write components to can be determined from FSIPath of ref.
	if ref.FSIPath == "" {
		return fsi.ErrNoLink
	}

	// Write components of the dataset to the working directory
	if err = fsi.WriteComponents(p.Ds, ref.FSIPath); err != nil {
		return err
	}

	*res, err = m.inst.fsi.Status(ctx, ref.FSIPath)
	return err
}

// RestoreParams provides parameters to the restore method.
type RestoreParams struct {
	Dir       string
	Ref       string
	Component string
}

// Restore method restores a component or all of the component files of a dataset from the repo
func (m *FSIMethods) Restore(p *RestoreParams, out *string) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.Restore", p, out)
	}
	ctx := context.TODO()

	if p.Ref == "" {
		return repo.ErrEmptyRef
	}
	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return fmt.Errorf("'%s' is not a valid dataset reference", p.Ref)
	}
	err = repo.CanonicalizeDatasetRef(m.inst.node.Repo, &ref)
	if err != nil && err != repo.ErrNoHistory {
		return
	}

	// Directory to write components to can be determined from FSIPath of ref.
	if p.Dir == "" && ref.FSIPath != "" {
		p.Dir = ref.FSIPath
	}
	// TODO(dlong): Perhaps disallow empty Dir (without FSIPath override), since relative
	// paths cause problems. Test using `qri connect`.

	ds := &dataset.Dataset{}

	if ref.Path != "" {
		// Read the previous version of the dataset from the repo
		ds, err = dsfs.LoadDataset(ctx, m.inst.node.Repo.Store(), ref.Path)
		if err != nil {
			return fmt.Errorf("loading dataset: %s", err)
		}
		if err = base.OpenDataset(ctx, m.inst.node.Repo.Filesystem(), ds); err != nil {
			return
		}
	}

	current, currFileMap, _, err := fsi.ReadDir(p.Dir)
	if err != nil {
		return err
	}

	removeComponents := []string{}

	history := &dataset.Dataset{
		Structure: ds.Structure,
	}

	if p.Component == "" {
		// Entire dataset.
		history.Assign(ds)
	} else if p.Component == "meta" {
		// Meta component.
		history.Meta = &dataset.Meta{}
		history.Meta.Assign(ds.Meta)
		if current.Meta != nil && !current.Meta.IsEmpty() && (ds.Meta == nil || ds.Meta.IsEmpty()) {
			removeComponents = append(removeComponents, "meta")
		}
	} else if p.Component == "structure" {
		history.Structure.Assign(ds.Structure)
		if !current.Structure.IsEmpty() && ds.Structure != nil {
			removeComponents = append(removeComponents, "structure")
		}
	} else if p.Component == "schema" || p.Component == "structure.schema" {
		// Schema is not a "real" component, is short for the structure's schema.
		if ds.Structure != nil {
			history.Structure.Schema = ds.Structure.Schema
		}
		if len(current.Structure.Schema) > 0 && (ds.Structure == nil || len(ds.Structure.Schema) == 0) {
			removeComponents = append(removeComponents, "schema")
		}
	} else if p.Component == "body" {
		// Body of the dataset.
		// This check for ref.Path is equivilant to making sure there's a previous version.
		if ref.Path != "" {
			df, err := dataset.ParseDataFormatString(history.Structure.Format)
			if err != nil {
				return err
			}
			fcfg, err := dataset.ParseFormatConfigMap(df, map[string]interface{}{})
			if err != nil {
				return err
			}
			bufData, err := actions.GetBody(m.inst.node, ds, df, fcfg, -1, -1, true)
			if err != nil {
				return err
			}
			history.SetBodyFile(qfs.NewMemfileBytes("body", bufData))
		} else {
			removeComponents = append(removeComponents, "body")
		}
	} else {
		return fmt.Errorf("Unknown component name \"%s\"", p.Component)
	}

	// Delete components that exist in the working directory but did not exist in previous version.
	if err = fsi.DeleteComponents(removeComponents, currFileMap, p.Dir); err != nil {
		return err
	}

	// Write components of the dataset to the working directory.
	return fsi.WriteComponents(history, p.Dir)
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

	*name, err = m.inst.fsi.InitDataset(*p)
	return err
}
