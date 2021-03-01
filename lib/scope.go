package lib

import (
	"context"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/repo"
)

// Scope represents the lifetime of a method call, abstractly connected to the caller of
// that method, such that the implementation is unaware of how it has been invoked.
// Using Scope instead of the global data on Instance lets us control user identity,
// permissions, and configuration, while also setting us up to properly run multiple
// operations at the same time to support multi-tenancy and multi-processing.
type Scope struct {
	ctx  context.Context
	inst *Instance
	// TODO(dustmop): Additional information, such as user identity, their profile, keys
}

// NewScope constructs a new Scope
func NewScope(ctx context.Context, inst *Instance) *Scope {
	return &Scope{
		ctx:  ctx,
		inst: inst,
	}
}

// Context returns the context for this scope. Though this pattern is usually discouraged,
// we're following http.Request's lead, as Scope plays the same role. The lifetime of a
// single Scope mathes the lifetime of the Context, this ownership is not long-lived
func (s *Scope) Context() context.Context {
	return s.ctx
}

// FSISubsystem returns a reference to the FSI subsystem
// TODO(dustmop): This subsystem contains global data, we should move that data out and
// into scope
func (s *Scope) FSISubsystem() *fsi.FSI {
	return s.inst.fsi
}

// Bus returns the event bus
func (s *Scope) Bus() event.Bus {
	// TODO: Filter only events for this scope.
	return s.inst.bus
}

// Filesystem returns a filesystem
func (s *Scope) Filesystem() qfs.Filesystem {
	return s.inst.qfs
}

// Dscache returns the dscache
func (s *Scope) Dscache() *dscache.Dscache {
	return s.inst.Dscache()
}

// ParseAndResolveRef parses a reference and resolves it
func (s *Scope) ParseAndResolveRef(ctx context.Context, refStr, source string) (dsref.Ref, string, error) {
	return s.inst.ParseAndResolveRef(ctx, refStr, source)
}

// ParseAndResolveRefWithWorkingDir parses a reference and resolves it with FSI info attached
func (s *Scope) ParseAndResolveRefWithWorkingDir(ctx context.Context, refstr, source string) (dsref.Ref, string, error) {
	return s.inst.ParseAndResolveRefWithWorkingDir(ctx, refstr, source)
}

// LoadDataset loads a dataset
func (s *Scope) LoadDataset(ctx context.Context, ref dsref.Ref, source string) (*dataset.Dataset, error) {
	return s.inst.LoadDataset(ctx, ref, source)
}

// Loader returns the default dataset ref loader
func (s *Scope) Loader() dsref.ParseResolveLoad {
	return NewParseResolveLoadFunc("", s.inst.defaultResolver(), s.inst)
}

// GetVersionInfoShim is in the process of being removed. Try not to add new callers.
func (s *Scope) GetVersionInfoShim(ref dsref.Ref) (*dsref.VersionInfo, error) {
	r := s.inst.Repo()
	return repo.GetVersionInfoShim(r, ref)
}
