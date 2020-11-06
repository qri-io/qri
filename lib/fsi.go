package lib

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
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

// LinkParams encapsulate parameters to the link method
type LinkParams struct {
	Dir string
	Ref string
}

// CreateLink creates a connection between a working drirectory and a dataset history
func (m *FSIMethods) CreateLink(p *LinkParams, res *dsref.VersionInfo) (err error) {
	// absolutize path name
	path, err := filepath.Abs(p.Dir)
	if err != nil {
		return err
	}
	p.Dir = path

	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("FSIMethods.CreateLink", p, res))
	}

	ctx := context.TODO()
	ref, _, err := m.inst.ParseAndResolveRef(ctx, p.Ref, "local")
	if err != nil {
		return err
	}

	res, _, err = m.inst.fsi.CreateLink(p.Dir, ref)
	return err
}

// Unlink removes the connection between a working directory and a dataset. If given only a
// directory, will remove the link file from that directory. If given only a reference,
// will remove the fsi path from that reference, and remove the link file from that fsi path
func (m *FSIMethods) Unlink(p *LinkParams, res *string) (err error) {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("FSIMethods.Unlink", p, res))
	}
	ctx := context.TODO()

	if p.Dir != "" && p.Ref != "" {
		return fmt.Errorf("Unlink should be called with either Dir or Ref, not both")
	}

	var ref dsref.Ref
	if p.Dir == "" {
		// If only ref provided, canonicalize it to get its ref
		ref, _, err = m.inst.ParseAndResolveRef(ctx, p.Ref, "local")
		if err != nil {
			return err
		}
		vi, err := repo.GetVersionInfoShim(m.inst.repo, ref)
		if err != nil {
			return err
		}
		if vi.FSIPath == "" {
			return fmt.Errorf("%s is not linked to a working directory", ref.Human())
		}
		p.Dir = vi.FSIPath
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
		return checkRPCError(m.inst.rpc.Call("FSIMethods.Status", dir, res))
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
		return checkRPCError(m.inst.rpc.Call("FSIMethods.AliasStatus", alias, res))
	}
	ctx := context.TODO()

	// If only ref provided, canonicalize it to get its ref
	ref, err := dsref.ParseHumanFriendly(*alias)
	if err != nil {
		return err
	}
	vi, err := repo.GetVersionInfoShim(m.inst.repo, ref)
	if err != nil {
		return err
	}

	*res, err = m.inst.fsi.Status(ctx, vi.FSIPath)
	return err
}

// WhatChanged gets changes that happened at a particular version in the history of the given
// dataset reference. Not used for FSI.
func (m *FSIMethods) WhatChanged(refstr *string, res *[]StatusItem) (err error) {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("FSIMethods.WhatChanged", refstr, res))
	}
	ctx := context.TODO()

	ref, _, err := m.inst.ParseAndResolveRef(ctx, *refstr, "local")
	if err != nil {
		return err
	}

	*res, err = m.inst.fsi.StatusAtVersion(ctx, ref)
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
		return checkRPCError(m.inst.rpc.Call("FSIMethods.Checkout", p, out))
	}
	ctx := context.TODO()

	// Require a non-empty, absolute path for the checkout
	if p.Dir == "" || !filepath.IsAbs(p.Dir) {
		return fmt.Errorf("need Dir to be a non-empty, absolute path")
	}

	log.Debugf("Checkout started, stat'ing %q", p.Dir)

	// If directory exists, error.
	if _, err = os.Stat(p.Dir); !os.IsNotExist(err) {
		return fmt.Errorf("directory with name \"%s\" already exists", p.Dir)
	}

	// Handle the ref to checkout
	ref, source, err := m.inst.ParseAndResolveRef(ctx, p.Ref, "")
	if err != nil {
		return err
	}

	if source != "" {
		return fmt.Errorf("auto-adding on checkout is not yet supported, please run `qri add %q` first", ref.Human())
	}

	log.Debugf("Checkout for ref %q", ref)

	// Fail early if link already exists
	if err := m.inst.fsi.EnsureRefNotLinked(ref); err != nil {
		return err
	}

	// Load dataset that is being checked out.
	ds, err := m.inst.LoadDataset(ctx, ref, "")
	if err != nil {
		log.Debugf("Checkout, dsfs.LoadDataset failed, error: %s", err)
		return err
	}

	// Create a directory.
	if err := os.Mkdir(p.Dir, os.ModePerm); err != nil {
		log.Debugf("Checkout, Mkdir failed, error: %s", ref)
		return err
	}
	log.Debugf("Checkout made directory %q", p.Dir)

	// Create the link file, containing the dataset reference.
	if _, _, err = m.inst.fsi.CreateLink(p.Dir, ref); err != nil {
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
		return checkRPCError(m.inst.rpc.Call("FSIMethods.Write", p, res))
	}
	ctx := context.TODO()

	ref, err := dsref.ParseHumanFriendly(p.Ref)
	if err != nil {
		return err
	}
	if p.Ds == nil {
		return fmt.Errorf("dataset is required")
	}

	datasetRef := reporef.RefFromDsref(ref)
	err = repo.CanonicalizeDatasetRef(m.inst.node.Repo, &datasetRef)
	if err != nil && err != repo.ErrNoHistory {
		return err
	}

	// Directory to write components to can be determined from FSIPath of ref.
	if datasetRef.FSIPath == "" {
		return fsi.ErrNoLink
	}

	// Write components of the dataset to the working directory
	err = fsi.WriteComponents(p.Ds, datasetRef.FSIPath, m.inst.node.Repo.Filesystem())
	if err != nil {
		return err
	}

	*res, err = m.inst.fsi.Status(ctx, datasetRef.FSIPath)
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
		return checkRPCError(m.inst.rpc.Call("FSIMethods.Restore", p, out))
	}
	ctx := context.TODO()

	ref, _, err := m.inst.ParseAndResolveRef(ctx, p.Ref, "local")
	if err != nil {
		return err
	}

	if p.Dir == "" {
		fsiRef := ref.Copy()
		if err = m.inst.fsi.ResolvedPath(&fsiRef); err != nil {
			return err
		}
		p.Dir = fsi.FilesystemPathToLocal(fsiRef.Path)
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

// InitDatasetParams proxies parameters to initialization
type InitDatasetParams = fsi.InitParams

// InitDataset creates a new dataset in a working directory
func (m *FSIMethods) InitDataset(p *InitDatasetParams, refstr *string) (err error) {
	if err = qfs.AbsPath(&p.BodyPath); err != nil {
		return err
	}

	if p.TargetDir == "" {
		p.TargetDir = "."
	}
	if err = qfs.AbsPath(&p.TargetDir); err != nil {
		return err
	}

	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("FSIMethods.InitDataset", p, refstr))
	}

	// If the dscache doesn't exist yet, it will only be created if the appropriate flag enables it.
	if p.UseDscache {
		m.inst.Dscache().CreateNewEnabled = true
	}

	ref, err := m.inst.fsi.InitDataset(*p)
	*refstr = ref.Human()
	return err
}

// CanInitDatasetWorkDir returns nil if the directory can init a dataset, or an error if not
func (m *FSIMethods) CanInitDatasetWorkDir(p *InitDatasetParams, ok *bool) error {
	targetPath := p.TargetDir
	bodyPath := p.BodyPath
	return m.inst.fsi.CanInitDatasetWorkDir(targetPath, bodyPath)
}

// EnsureParams holds values for EnsureRef call
type EnsureParams struct {
	Dir string
	Ref string
}

// EnsureRef will modify the directory path in the repo for the given reference
func (m *FSIMethods) EnsureRef(p *EnsureParams, out *dsref.VersionInfo) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("FSIMethods.EnsureRef", p, out))
	}

	ref, err := dsref.Parse(p.Ref)
	if err != nil {
		return err
	}

	vi, err := m.inst.fsi.ModifyLinkDirectory(p.Dir, ref)
	*out = *vi
	return err
}

// PathJoinPosix joins two paths, and makes it explicitly clear we want POSIX slashes
func PathJoinPosix(left, right string) string {
	return path.Join(left, right)
}
