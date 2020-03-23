package lib

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
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
func (m *FSIMethods) LinkedRefs(p *ListParams, res *[]reporef.DatasetRef) (err error) {
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

	*res, _, err = m.inst.fsi.CreateLink(p.Dir, p.Ref)
	return err
}

// Unlink removes the connection between a working directory and a dataset. If given only a
// directory, will remove the link file from that directory. If given only a reference,
// will remove the fsi path from that reference, and remove the link file from that fsi path
func (m *FSIMethods) Unlink(p *LinkParams, res *string) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.Unlink", p, res)
	}

	if p.Dir != "" && p.Ref != "" {
		return fmt.Errorf("Unlink should be called with either Dir or Ref, not both")
	}

	var ref dsref.Ref
	if p.Dir == "" {
		// If only ref provided, canonicalize it to get its ref
		datasetRef, err := repo.ParseDatasetRef(p.Ref)
		if err != nil {
			return err
		}
		if err = repo.CanonicalizeDatasetRef(m.inst.repo, &datasetRef); err != nil {
			if err != repo.ErrNoHistory {
				return err
			}
		}
		if datasetRef.FSIPath == "" {
			return fmt.Errorf("%s is not linked to a directory", p.Ref)
		}
		p.Dir = datasetRef.FSIPath
		ref = reporef.ConvertToDsref(datasetRef)
	}

	if err := m.inst.fsi.Unlink(p.Dir, ref); err != nil {
		return err
	}

	*res = ref.Alias()
	return nil
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

// WhatChanged gets changes that happened at a particular version in the history of the given
// dataset reference. Not used for FSI.
func (m *FSIMethods) WhatChanged(ref *string, res *[]StatusItem) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.WhatChanged", ref, res)
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

	log.Debugf("Checkout started, stat'ing %q", p.Dir)

	// TODO(dlong): Fail if Dir is "", should be required to specify a location. Should probably
	// only allow absolute paths. Add tests.

	// If directory exists, error.
	if _, err = os.Stat(p.Dir); !os.IsNotExist(err) {
		return fmt.Errorf("directory with name \"%s\" already exists", p.Dir)
	}

	// Handle the ref to checkout.
	ref := &reporef.DatasetRef{}
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

	log.Debugf("Checkout for ref %q", ref)

	// Load dataset that is being checked out.
	ds, err := dsfs.LoadDataset(ctx, m.inst.repo.Store(), ref.Path)
	if err != nil {
		log.Debugf("Checkout, dsfs.LoadDataset failed, error: %s", err)
		return fmt.Errorf("error loading dataset")
	}
	ds.Name = ref.Name
	ds.Peername = ref.Peername
	if err = base.OpenDataset(ctx, m.inst.repo.Filesystem(), ds); err != nil {
		log.Debugf("Checkout, base.OpenDataset failed, error: %s", ref)
		return
	}
	log.Debugf("Checkout loaded dataset %q", ref)

	// Create a directory.
	if err := os.Mkdir(p.Dir, os.ModePerm); err != nil {
		log.Debugf("Checkout, Mkdir failed, error: %s", ref)
		return err
	}
	log.Debugf("Checkout made directory %q", p.Dir)

	// Create the link file, containing the dataset reference.
	if _, _, err = m.inst.fsi.CreateLink(p.Dir, p.Ref); err != nil {
		log.Debugf("Checkout, fsi.CreateLink failed, error: %s", ref)
		return err
	}
	log.Debugf("Checkout created link for %q <-> %q", p.Dir, p.Ref)

	// Write components of the dataset to the working directory.
	err = fsi.WriteComponents(ds, p.Dir, m.inst.node.Repo.Filesystem())
	if err != nil {
		log.Debugf("Checkout, fsi.WriteComponents failed, error: %s", ref)
	}
	log.Debugf("Checkout wrote components, successfully checked out dataset")

	log.Debugf("Checkout successfully checked out dataset")
	return nil
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
	if err = fsi.WriteComponents(p.Ds, ref.FSIPath, m.inst.node.Repo.Filesystem()); err != nil {
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
	if p.Dir == "" {
		return fmt.Errorf("no FSIPath or Dir given")
	}

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

	// Build component container from the dataset from the repo.
	repoContainer := component.ConvertDatasetToComponents(ds, m.inst.node.Repo.Filesystem())
	repoContainer.Base().RemoveSubcomponent("commit")
	repoContainer.DropDerivedValues()

	// Build component container from FSI directory.
	diskContainer, err := component.ListDirectoryComponents(p.Dir)
	if err != nil {
		return err
	}
	err = component.ExpandListedComponents(diskContainer, m.inst.node.Repo.Filesystem())
	if err != nil {
		return err
	}

	for _, compName := range component.AllSubcomponentNames() {
		if p.Component == "" || p.Component == compName {
			if repoContainer.Base().GetSubcomponent(compName) == nil {
				fsi.DeleteComponent(diskContainer, compName, p.Dir)
			} else {
				fsi.WriteComponent(repoContainer, compName, p.Dir)
			}
		}
	}
	return nil
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

// EnsureParams holds values for EnsureRef call
type EnsureParams struct {
	Dir string
	Ref string
}

// EnsureRef will modify the directory path in the repo for the given reference
func (m *FSIMethods) EnsureRef(p *EnsureParams, out *bool) error {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("FSIMethods.EnsureRef", p, out)
	}

	return m.inst.fsi.ModifyLinkDirectory(p.Dir, p.Ref)
}
