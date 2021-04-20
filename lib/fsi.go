package lib

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/repo"
)

// FSIMethods groups together methods for FSI
type FSIMethods struct {
	d dispatcher
}

// Name returns the name of this method group
func (m FSIMethods) Name() string {
	return "fsi"
}

// Attributes defines attributes for each method
func (m FSIMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"status":                {Endpoint: AEStatus, HTTPVerb: "POST"},
		"caninitdatasetworkdir": {Endpoint: AECanInitDatasetWorkDir, HTTPVerb: "POST"},
		"init":                  {Endpoint: AEInit, HTTPVerb: "POST"},
		"checkout":              {Endpoint: AECheckout, HTTPVerb: "POST"},
		"ensureref":             {Endpoint: AEEnsureRef, HTTPVerb: "POST"},
		"restore":               {Endpoint: AERestore, HTTPVerb: "POST"},
		"write":                 {Endpoint: AEFSIWrite, HTTPVerb: "POST"},
		"createlink":            {Endpoint: AEFSICreateLink, HTTPVerb: "POST"},
		"unlink":                {Endpoint: AEFSIUnlink, HTTPVerb: "POST"},
	}
}

// LinkParams encapsulate parameters for linked datasets
type LinkParams struct {
	Dir string `qri:"fspath"`
	Ref string
}

// FSIWriteParams encapsultes arguments for writing to an FSI-linked directory
type FSIWriteParams struct {
	Ref     string
	Dataset *dataset.Dataset
}

// RestoreParams provides parameters to the restore method.
type RestoreParams struct {
	Dir      string `qri:"fspath"`
	Ref      string
	Version  string
	Selector string
}

// InitDatasetParams proxies parameters to initialization
type InitDatasetParams = fsi.InitParams

// StatusItem is an alias for an fsi.StatusItem
type StatusItem = fsi.StatusItem

// CreateLink creates a connection between a working directory and a dataset history
func (m FSIMethods) CreateLink(ctx context.Context, p *LinkParams) (*dsref.VersionInfo, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "createlink"), p)
	if res, ok := got.(*dsref.VersionInfo); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Unlink removes the connection between a working directory and a dataset. If given only a
// directory, will remove the link file from that directory. If given only a reference,
// will remove the fsi path from that reference, and remove the link file from that fsi path
func (m FSIMethods) Unlink(ctx context.Context, p *LinkParams) (string, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "unlink"), p)
	if res, ok := got.(string); ok {
		return res, err
	}
	return "", dispatchReturnError(got, err)
}

// Status checks for any modifications or errors in a linked directory against its previous
// version in the repo. Must only be called if FSI is enabled for this dataset.
func (m FSIMethods) Status(ctx context.Context, p *LinkParams) ([]StatusItem, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "status"), p)
	if res, ok := got.([]StatusItem); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Checkout method writes a dataset to a directory as individual files.
func (m FSIMethods) Checkout(ctx context.Context, p *LinkParams) error {
	_, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "checkout"), p)
	return err
}

// Write mutates a linked dataset on the filesystem
func (m FSIMethods) Write(ctx context.Context, p *FSIWriteParams) ([]StatusItem, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "write"), p)
	if res, ok := got.([]StatusItem); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Restore method restores a component or all of the component files of a dataset from the repo
func (m FSIMethods) Restore(ctx context.Context, p *RestoreParams) error {
	_, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "restore"), p)
	return err
}

// Init initializes a new working directory for a linked dataset
func (m FSIMethods) Init(ctx context.Context, p *InitDatasetParams) (string, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "init"), p)
	if res, ok := got.(string); ok {
		return res, err
	}
	return "", dispatchReturnError(got, err)
}

// CanInitDatasetWorkDir returns nil if the directory can init a dataset, or an error if not
func (m FSIMethods) CanInitDatasetWorkDir(ctx context.Context, p *InitDatasetParams) error {
	_, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "caninitdatasetworkdir"), p)
	return err
}

// EnsureRef will modify the directory path in the repo for the given reference
func (m FSIMethods) EnsureRef(ctx context.Context, p *LinkParams) (*dsref.VersionInfo, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "ensureref"), p)
	if res, ok := got.(*dsref.VersionInfo); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Implementations for FSI methods follow

// fsiImpl holds the method implementations for FSI
type fsiImpl struct{}

// CreateLink creates a connection between a working directory and a dataset history
func (fsiImpl) CreateLink(scope scope, p *LinkParams) (*dsref.VersionInfo, error) {
	ctx := scope.Context()

	ref, _, err := scope.ParseAndResolveRef(ctx, p.Ref)
	if err != nil {
		return nil, err
	}
	vinfo, _, err := scope.FSISubsystem().CreateLink(ctx, p.Dir, ref)
	return vinfo, err
}

// Unlink removes the connection between a working directory and a dataset. If given only a
// directory, will remove the link file from that directory. If given only a reference,
// will remove the fsi path from that reference, and remove the link file from that fsi path
func (fsiImpl) Unlink(scope scope, p *LinkParams) (string, error) {
	ctx := scope.Context()

	if p.Dir != "" && p.Ref != "" {
		return "", fmt.Errorf("Unlink should be called with either Dir or Ref, not both")
	}

	var ref dsref.Ref
	if p.Dir == "" {
		// If only ref provided, canonicalize it to get its ref
		var err error
		ref, _, err = scope.ParseAndResolveRef(ctx, p.Ref)
		if err != nil {
			return "", err
		}
		// NOTE: GetVersionInfoShim is in the process of being removed. Try not to add
		// new callers.
		vi, err := scope.GetVersionInfoShim(ref)
		if err != nil {
			return "", err
		}
		if vi.FSIPath == "" {
			return "", fmt.Errorf("%s is not linked to a working directory", ref.Human())
		}
		p.Dir = vi.FSIPath
	}

	if err := scope.FSISubsystem().Unlink(ctx, p.Dir, ref); err != nil {
		return "", err
	}

	return ref.Alias(), nil
}

// Status checks for any modifications or errors in a linked directory against its previous
// version in the repo. Must only be called if FSI is enabled for this dataset.
func (fsiImpl) Status(scope scope, p *LinkParams) ([]StatusItem, error) {
	ctx := scope.Context()

	if p.Dir == "" && p.Ref == "" {
		return nil, fmt.Errorf("either Dir or Ref required for status")
	}

	// If the directory is given, get the status of the linked dataset
	if p.Dir != "" {
		return scope.FSISubsystem().Status(ctx, p.Dir)
	}

	// Otherwise, get the file system path by looking up the ref
	ref, err := dsref.ParseHumanFriendly(p.Ref)
	if err != nil {
		return nil, err
	}
	vi, err := scope.GetVersionInfoShim(ref)
	if err != nil {
		return nil, err
	}

	return scope.FSISubsystem().Status(ctx, vi.FSIPath)
}

// Checkout method writes a dataset to a directory as individual files.
func (fsiImpl) Checkout(scope scope, p *LinkParams) error {
	ctx := scope.Context()

	// Require a non-empty, absolute path for the checkout
	if p.Dir == "" || !filepath.IsAbs(p.Dir) {
		return fmt.Errorf("need Dir to be a non-empty, absolute path")
	}

	log.Debugf("Checkout started, stat'ing %q", p.Dir)

	// If directory exists, error.
	if _, err := os.Stat(p.Dir); !os.IsNotExist(err) {
		return fmt.Errorf("directory with name \"%s\" already exists", p.Dir)
	}

	ds, err := scope.Loader().LoadDataset(ctx, p.Ref)
	if err != nil {
		log.Debugf("Checkout, dsfs.LoadDataset failed, error: %s", err)
		return err
	}

	ref := dsref.ConvertDatasetToVersionInfo(ds).SimpleRef()

	// Fail early if link already exists
	if err := scope.FSISubsystem().EnsureRefNotLinked(ref); err != nil {
		return err
	}

	// Create a directory.
	if err := os.Mkdir(p.Dir, os.ModePerm); err != nil {
		log.Debugf("Checkout, Mkdir failed, error: %s", ref)
		return err
	}
	log.Debugf("Checkout made directory %q", p.Dir)

	// Create the link file, containing the dataset reference.
	if _, _, err = scope.FSISubsystem().CreateLink(ctx, p.Dir, ref); err != nil {
		log.Debugf("Checkout, fsi.CreateLink failed, error: %s", ref)
		return err
	}
	log.Debugf("Checkout created link for %q <-> %q", p.Dir, p.Ref)

	// Write components of the dataset to the working directory.
	err = fsi.WriteComponents(ds, p.Dir, scope.Filesystem())
	if err != nil {
		log.Debugf("Checkout, fsi.WriteComponents failed, error: %s", ref)
	}
	log.Debugf("Checkout wrote components, successfully checked out dataset")

	log.Debugf("Checkout successfully checked out dataset")
	return nil
}

// Write mutates a linked dataset on the filesystem
func (fsiImpl) Write(scope scope, p *FSIWriteParams) ([]StatusItem, error) {
	ctx := scope.Context()

	if p.Dataset == nil {
		return nil, fmt.Errorf("dataset is required")
	}

	ref, _, err := scope.ParseAndResolveRef(ctx, p.Ref)
	if err != nil {
		return nil, err
	}

	vi, err := scope.GetVersionInfoShim(ref)
	if err != nil && err != repo.ErrNoHistory {
		return nil, err
	}

	// Directory to write components to can be determined from FSIPath of versionInfo
	if vi.FSIPath == "" {
		return nil, fsi.ErrNoLink
	}

	// Write components of the dataset to the working directory
	err = fsi.WriteComponents(p.Dataset, vi.FSIPath, scope.Filesystem())
	if err != nil {
		return nil, err
	}

	return scope.FSISubsystem().Status(ctx, vi.FSIPath)
}

// Restore method restores a component or all of the component files of a dataset from the repo
func (fsiImpl) Restore(scope scope, p *RestoreParams) error {
	ctx := scope.Context()

	ref, _, err := scope.ParseAndResolveRef(ctx, p.Ref)
	if err != nil {
		return err
	}

	if p.Version != "" {
		ref.Path = p.Version
	}

	if p.Dir == "" {
		fsiRef := ref.Copy()
		if err = scope.FSISubsystem().ResolvedPath(&fsiRef); err != nil {
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
		ds, err = dsfs.LoadDataset(ctx, scope.Filesystem(), ref.Path)
		if err != nil {
			return fmt.Errorf("loading dataset: %s", err)
		}
		if err = base.OpenDataset(ctx, scope.Filesystem(), ds); err != nil {
			return err
		}
	}

	// Build component container from the dataset from the repo.
	repoContainer := component.ConvertDatasetToComponents(ds, scope.Filesystem())
	repoContainer.Base().RemoveSubcomponent("commit")
	repoContainer.DropDerivedValues()

	// Build component container from FSI directory.
	diskContainer, err := component.ListDirectoryComponents(p.Dir)
	if err != nil {
		return err
	}
	err = component.ExpandListedComponents(diskContainer, scope.Filesystem())
	if err != nil {
		return err
	}

	for _, compName := range component.AllSubcomponentNames() {
		if p.Selector == "" || p.Selector == compName {
			if repoContainer.Base().GetSubcomponent(compName) == nil {
				fsi.DeleteComponent(diskContainer, compName, p.Dir)
			} else {
				fsi.WriteComponent(repoContainer, compName, p.Dir)
			}
		}
	}
	return nil
}

// Init creates a new dataset in a working directory
func (fsiImpl) Init(scope scope, p *InitDatasetParams) (string, error) {
	ctx := scope.Context()

	if p.UseDscache {
		scope.Dscache().CreateNewEnabled = true
	}
	ref, err := scope.FSISubsystem().InitDataset(ctx, *p)
	refstr := ref.Human()
	return refstr, err
}

// CanInitDatasetWorkDir returns nil if the directory can init a dataset, or an error if not
func (fsiImpl) CanInitDatasetWorkDir(scope scope, p *InitDatasetParams) error {
	return scope.FSISubsystem().CanInitDatasetWorkDir(p.TargetDir, p.BodyPath)
}

// EnsureRef will modify the directory path in the repo for the given reference
func (fsiImpl) EnsureRef(scope scope, p *LinkParams) (*dsref.VersionInfo, error) {
	ctx := scope.Context()

	ref, err := dsref.Parse(p.Ref)
	if err != nil {
		return nil, err
	}

	return scope.FSISubsystem().ModifyLinkDirectory(ctx, p.Dir, ref)
}

// PathJoinPosix joins two paths, and makes it explicitly clear we want POSIX slashes
func PathJoinPosix(left, right string) string {
	return path.Join(left, right)
}
