package lib

import (
	"context"

	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/registry/regclient"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/stats"
)

// scope represents the lifetime of a method call, abstractly connected to the caller of
// that method, such that the implementation is unaware of how it has been invoked.
// Using scope instead of the global data on Instance lets us control user identity,
// permissions, and configuration, while also setting us up to properly run multiple
// operations at the same time to support multi-tenancy and multi-processing.
type scope struct {
	ctx    context.Context
	inst   *Instance
	pro    *profile.Profile
	source string
	// TODO(dustmop): Additional information, such as user identity, their profile, keys
}

func newScope(ctx context.Context, inst *Instance, source string) (scope, error) {
	pro, err := inst.activeProfile(ctx)
	if err != nil {
		return scope{}, err
	}

	return scope{
		ctx:    ctx,
		inst:   inst,
		pro:    pro,
		source: source,
	}, nil
}

func (s *scope) ActiveProfile() *profile.Profile {
	return s.pro
}

// Bus returns the event bus
func (s *scope) Bus() event.Bus {
	// TODO(dustmop): Filter only events for this scope.
	return s.inst.bus
}

// ChangeConfig implements the ConfigSetter interface
func (s *scope) ChangeConfig(ctg *config.Config) error {
	return s.inst.ChangeConfig(ctg)
}

// Config returns the config
func (s *scope) Config() *config.Config {
	return s.inst.cfg
}

// Context returns the context for this scope. Though this pattern is usually discouraged,
// we're following http.Request's lead, as scope plays the same role. The lifetime of a
// single scope matches the lifetime of the Context; this ownership is not long-lived
func (s *scope) Context() context.Context {
	return s.ctx
}

// AppContext returns the context of the top-level application, to enable graceful shutdowns
func (s *scope) AppContext() context.Context {
	return s.inst.appCtx
}

// Dscache returns the dscache
func (s *scope) Dscache() *dscache.Dscache {
	return s.inst.Dscache()
}

// FSISubsystem returns a reference to the FSI subsystem
// TODO(dustmop): This subsystem contains global data, we should move that data out and
// into scope
func (s *scope) FSISubsystem() *fsi.FSI {
	return s.inst.fsi
}

// Filesystem returns a filesystem
func (s *scope) Filesystem() *muxfs.Mux {
	return s.inst.qfs
}

// GetVersionInfoShim is in the process of being removed. Try not to add new callers.
func (s *scope) GetVersionInfoShim(ref dsref.Ref) (*dsref.VersionInfo, error) {
	r := s.inst.Repo()
	return repo.GetVersionInfoShim(r, ref)
}

// Loader returns a loader that can load datasets
func (s *scope) Loader() dsref.Loader {
	return &datasetLoader{s.inst, s.source}
}

// Logbook returns the repo logbook
func (s *scope) Logbook() *logbook.Book {
	return s.inst.logbook
}

// Node returns the p2p.QriNode
func (s *scope) Node() *p2p.QriNode {
	return s.inst.node
}

// ParseAndResolveRef parses a reference and resolves it
func (s *scope) ParseAndResolveRef(ctx context.Context, refStr string) (dsref.Ref, string, error) {
	return s.inst.ParseAndResolveRef(ctx, refStr, s.source)
}

// ParseAndResolveRefWithWorkingDir parses a reference and resolves it with FSI info attached
func (s *scope) ParseAndResolveRefWithWorkingDir(ctx context.Context, refstr string) (dsref.Ref, string, error) {
	return s.inst.ParseAndResolveRefWithWorkingDir(ctx, refstr, s.source)
}

// Profiles accesses the profile store
func (s *scope) Profiles() profile.Store {
	return s.inst.profiles
}

// RegistryClient returns a client that can send requests to the registry
func (s *scope) RegistryClient() *regclient.Client {
	return s.inst.registry
}

// RemoteClient exposes the instance client for making requests to remotes
func (s *scope) RemoteClient() remote.Client {
	return s.inst.remoteClient
}

// Repo returns the repo store
func (s *scope) Repo() repo.Repo {
	return s.inst.repo
}

// RepoPath returns the path to the repo
func (s *scope) RepoPath() string {
	return s.inst.repoPath
}

// ResolveReference finds the identifier & HEAD path for a dataset reference.
// the mode parameter determines which subsystems of Qri to use when resolving
func (s *scope) ResolveReference(ctx context.Context, ref *dsref.Ref) (string, error) {
	return s.inst.ResolveReference(ctx, ref, s.source)
}

// SourceName returns the name of the source that is used for reference resolution
func (s *scope) SourceName() string {
	return s.source
}

// LocalResolver returns a resolver for local refs
func (s *scope) LocalResolver() (dsref.Resolver, error) {
	return s.inst.resolverForSource("local")
}

// SetLogbook allows you to replace the current logbook with the given logbook
func (s *scope) SetLogbook(l *logbook.Book) {
	s.inst.logbook = l
}

// SourceName is the name of the configured source for reference resolution
func (s *scope) SourceName() string {
	return s.source
}

// Stats returns the stats service
func (s *scope) Stats() *stats.Service {
	return s.inst.stats
}

// UseDscache returns whether dscache should be generated
// TODO(dustmop): Add a config option or environment variable to experimentally
// enable the dscache
func (s *scope) UseDscache() bool {
	return false
}
