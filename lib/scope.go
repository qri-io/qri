package lib

import (
	"context"

	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/automation"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/collection"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/registry/regclient"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/stats"
)

// scope represents the lifetime of a method call, abstractly connected to the
// caller of that method, such that the implementation is unaware of how it has
// been invoked. Using scope instead of the global data on Instance lets us
// control user identity, permissions, and configuration, while also setting us
// up to properly run multiple operations at the same time to support
// multi-tenancy and multi-processing.
type scope struct {
	ctx    context.Context
	inst   *Instance
	pro    *profile.Profile
	method string
	source string
}

func newScope(ctx context.Context, inst *Instance, method, source string) (scope, error) {
	pro, err := inst.activeProfile(ctx)
	if err != nil {
		return scope{}, err
	}

	// Add the profileID to the context to identify this user
	ctx = profile.AddIDToContext(ctx, pro.ID.Encode())
	return scope{
		ctx:    ctx,
		inst:   inst,
		method: method,
		pro:    pro,
		source: source,
	}, nil
}

func newScopeFromWorkflow(ctx context.Context, inst *Instance, wf *workflow.Workflow) (scope, error) {
	ctx = profile.AddIDToContext(ctx, wf.OwnerID.Encode())
	pro, err := inst.profiles.GetProfile(ctx, wf.OwnerID)
	if err != nil {
		log.Debugw("getting profile", "profileID", wf.OwnerID.Encode(), "err", err)
		return scope{}, err
	}

	return scope{
		ctx:    ctx,
		inst:   inst,
		pro:    pro,
		source: "local",
	}, nil
}

func (s *scope) ActiveProfile() *profile.Profile {
	return s.pro
}

// Orchestrator returns the automation orchestrator
func (s *scope) AutomationOrchestrator() *automation.Orchestrator {
	return s.inst.automation
}

// Bus returns the event bus
func (s *scope) Bus() event.Bus {
	// TODO(dustmop): Filter only events for this scope.
	return s.inst.bus
}

// sendEvent publishes an event with an identifier on the instance bus, using
// the scope context
func (s *scope) sendEvent(typ event.Type, id string, payload interface{}) error {
	err := s.inst.bus.PublishID(s.ctx, typ, id, payload)
	if err != nil {
		log.Debug(err)
	}
	return err
}

// ChangeConfig implements the ConfigSetter interface
func (s *scope) ChangeConfig(ctg *config.Config) error {
	return s.inst.ChangeConfig(ctg)
}

// Config returns the config
func (s *scope) Config() *config.Config {
	return s.inst.cfg
}

// Context returns the context for this scope. Though this pattern is usually
// discouraged, we're following http.Request's lead, as scope plays the same
// role. The lifetime of a single scope matches the lifetime of the Context;
// this ownership is not long-lived
func (s *scope) Context() context.Context {
	return s.ctx
}

// AppContext returns the context of the top-level application, to enable
// graceful shutdowns
func (s *scope) AppContext() context.Context {
	return s.inst.appCtx
}

// CollectionSet returns the set of collections
func (s *scope) CollectionSet() collection.Set {
	return s.inst.collections
}

// ReplaceParentContext returns a copy of the scope bound to a new parent
// context
func (s *scope) ReplaceParentContext(newParent context.Context) scope {
	// Copy values from the old context to the new one
	profileID := profile.IDFromCtx(s.ctx)
	if profileID != "" {
		newParent = profile.AddIDToContext(newParent, profileID)
	}
	// Return a copy of the scope, except the context is new
	return scope{
		ctx:    newParent,
		inst:   s.inst,
		pro:    s.pro,
		source: s.source,
	}
}

// Dscache returns the dscache
func (s *scope) Dscache() *dscache.Dscache {
	return s.inst.Dscache()
}

// Filesystem returns a filesystem
func (s *scope) Filesystem() *muxfs.Mux {
	return s.inst.qfs
}

// GetVersionInfoShim is in the process of being removed. Try not to add new
// callers.
func (s *scope) GetVersionInfoShim(ref dsref.Ref) (*dsref.VersionInfo, error) {
	r := s.inst.Repo()
	return repo.GetVersionInfoShim(r, ref)
}

// Loader returns a loader that can load datasets
func (s *scope) Loader() dsref.Loader {
	username := s.inst.cfg.Profile.Peername
	return newDatasetLoader(s.inst, username, s.source)
}

// Logbook returns the repo logbook
func (s *scope) Logbook() *logbook.Book {
	return s.inst.logbook
}

// MakeCursor returns a cursor that is able to retrieve the next page of results
func (s *scope) MakeCursor(numReturned int, nextPage interface{}) Cursor {
	if numReturned < 1 {
		return nil
	}
	return cursor{s.inst, s.method, nextPage}
}

// Node returns the p2p.QriNode
func (s *scope) Node() *p2p.QriNode {
	return s.inst.node
}

// ParseAndResolveRef parses a reference and resolves it
func (s *scope) ParseAndResolveRef(ctx context.Context, refStr string) (dsref.Ref, string, error) {
	return s.inst.ParseAndResolveRef(ctx, refStr, s.source)
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
