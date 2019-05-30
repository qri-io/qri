// Package lib implements core qri business logic. It exports
// canonical methods that a qri instance can perform regardless of
// client interface. API's of any sort must use lib methods
package lib

import (
	"context"
	"encoding/gob"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"sync"

	golog "github.com/ipfs/go-log"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/qri-io/dag"
	"github.com/qri-io/dag/dsync"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	ipfs "github.com/qri-io/qfs/cafs/ipfs"
	"github.com/qri-io/qfs/httpfs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/config/migrate"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	fsrepo "github.com/qri-io/qri/repo/fs"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/update"
	"github.com/qri-io/qri/update/cron"
	"github.com/qri-io/registry/regclient"
)

var (
	// defaultIPFSLocation is where qri data defaults to looking for / setting up
	// IPFS. The keyword $HOME will be replaced with the current user home
	// directory. only $HOME is replaced (no other $ env vars).
	defaultIPFSLocation = "$HOME/.ipfs"

	log = golog.Logger("lib")
)

// VersionNumber is the current version qri
const VersionNumber = "0.8.3-dev"

func init() {
	// Fields like dataset.Structure.Schema contain data of arbitrary types,
	// registering with the gob package prevents errors when sending them
	// over net/rpc calls.
	gob.Register([]interface{}{})
	gob.Register(map[string]interface{}{})
}

// Receivers returns a slice of CoreRequests that defines the full local
// API of lib methods
func Receivers(inst *Instance) []Methods {
	node := inst.Node()
	r := inst.Repo()

	return []Methods{
		NewDatasetRequests(node, nil),
		NewRegistryRequests(node, nil),
		NewLogRequests(node, nil),
		NewExportRequests(node, nil),
		NewPeerRequests(node, nil),
		NewProfileMethods(inst),
		NewConfigMethods(inst),
		NewSearchRequests(node, nil),
		NewRenderRequests(r, nil),
<<<<<<< HEAD
=======
		NewSelectionRequests(r, nil),
		NewRemoteMethods(inst),
>>>>>>> refactor(dsync): incorporate dsync updates
		NewUpdateMethods(inst),
		NewFSIMethods(inst),
	}
}

// Methods is a related set of library functions
type Methods interface {
	// CoreRequestsName confirms participation in the CoreRequests interface while
	// also giving a human readable string for logging purposes
	// TODO (b5): rename this interface to "MethodsName", or remove entirely
	CoreRequestsName() string
}

// InstanceOptions provides details to NewInstance.
// New will alter InstanceOptions by applying
// any provided Option functions
// to distinguish "Options" from "Config":
// * Options contains state that can only be determined at runtime
// * Config consists only of static values stored in a configuration file
// Options may override config in specific cases to avoid undefined state
type InstanceOptions struct {
	Cfg     *config.Config
	Streams ioes.IOStreams
}

// Option is a function that manipulates config details when fed to New(). Fields on
// the o parameter may be null, functions cannot assume the Config is non-null.
type Option func(o *InstanceOptions) error

// OptConfig supplies a configuration directly
func OptConfig(cfg *config.Config) Option {
	return func(o *InstanceOptions) error {
		o.Cfg = cfg
		return nil
	}
}

// OptSetIPFSPath configures the directory to read IPFS from (only if the
// configured store is IPFS). If the given path is the empty string, default
// to the standard IPFS config:
// * IPFS_PATH environment variable if set
// * if none set: "$HOME/.ipfs"
func OptSetIPFSPath(path string) Option {
	return func(o *InstanceOptions) error {
		if o.Cfg == nil {
			return fmt.Errorf("config is nil, can't set IPFS path")
		}
		if o.Cfg.Store == nil {
			return fmt.Errorf("config.Store is nil, can't check type")
		}
		if o.Cfg.Store.Type == "ipfs" && o.Cfg.Store.Path == "" {
			if path == "" {
				path = os.Getenv("IPFS_PATH")
				if path == "" {
					dir, err := homedir.Dir()
					if err != nil {
						return err
					}
					path = strings.Replace(defaultIPFSLocation, "$HOME", dir, 1)
				}
			}
			o.Cfg.Store.Path = path
		}
		return nil
	}
}

// OptIOStreams sets the input IOStreams
func OptIOStreams(streams ioes.IOStreams) Option {
	return func(o *InstanceOptions) error {
		o.Streams = streams
		return nil
	}
}

// OptStdIOStreams sets treams to std, stdout, & stderr
func OptStdIOStreams() Option {
	return func(o *InstanceOptions) error {
		o.Streams = ioes.NewStdIOStreams()
		return nil
	}
}

// OptCheckConfigMigrations checks for any configuration migrations that may need to be run
// running & updating config if so
func OptCheckConfigMigrations(cfgPath string) Option {
	return func(o *InstanceOptions) error {
		if o.Cfg == nil {
			return fmt.Errorf("no config file to check for migrations")
		}

		migrated, err := migrate.RunMigrations(o.Streams, o.Cfg)
		if err != nil {
			return err
		}

		if migrated {
			return o.Cfg.WriteToFile(cfgPath)
		}
		return nil
	}
}

// NewInstance creates a new Qri Instance, if no Option funcs are provided,
// New uses a default set of Option funcs. Any Option functions passed to this
// function must check whether their fields are nil or not.
func NewInstance(ctx context.Context, repoPath string, opts ...Option) (qri *Instance, err error) {
	if repoPath == "" {
		return nil, fmt.Errorf("repo path is required")
	}

	o := &InstanceOptions{}

	// attempt to load a base configuration from repoPath
	if o.Cfg, err = loadRepoConfig(repoPath); err != nil {
		return
	}

	if len(opts) == 0 {
		// default to a standard composition of Option funcs
		opts = []Option{
			OptStdIOStreams(),
			OptSetIPFSPath(""),
			OptCheckConfigMigrations(""),
		}
	}
	for _, opt := range opts {
		if err = opt(o); err != nil {
			return
		}
	}

	cfg := o.Cfg
	if cfg == nil {
		// If at this point we don't have a configuration pointer one couldn't be
		// loaded from repoPath, and a configuration wasn't provided through Options,
		// so qri needs to be set up
		err = fmt.Errorf("no qri repo found, please run `qri setup`")
		return
	} else if err = cfg.Validate(); err != nil {
		return
	}

	ctx, teardown := context.WithCancel(ctx)
	inst := &Instance{
		ctx:      ctx,
		teardown: teardown,
		repoPath: repoPath,
		cfg:      cfg,
		streams:  o.Streams,
	}
	qri = inst

	// configure logging straight away
	if cfg != nil && cfg.Logging != nil {
		for name, level := range cfg.Logging.Levels {
			golog.SetLogLevel(name, level)
		}
	}

	if inst.cron, err = newCron(cfg, inst.repoPath); err != nil {
		return nil, fmt.Errorf("newCron: %s", err)
	}

	// check if we're operating over RPC
	if cfg.RPC.Enabled {
		addr := fmt.Sprintf(":%d", cfg.RPC.Port)
		log.Infof("Dialing rpc address %s", addr)
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			// we have a connection
			inst.rpc = rpc.NewClient(conn)
			return qri, err
		}
	}

	if inst.store, err = newStore(ctx, cfg); err != nil {
		return nil, fmt.Errorf("newStore: %s", err)
	}
	if inst.qfs, err = newFilesystem(cfg, inst.store); err != nil {
		return nil, fmt.Errorf("newFilesystem: %s", err)
	}
	inst.registry = newRegClient(cfg)

	if inst.repo, err = newRepo(inst.repoPath, cfg, inst.store, inst.registry); err != nil {
		return nil, fmt.Errorf("newRepo: %s", err)
	}
	if qfssetter, ok := inst.repo.(repo.QFSSetter); ok {
		qfssetter.SetFilesystem(inst.qfs)
	}

	if inst.node, err = p2p.NewQriNode(inst.repo, cfg.P2P); err != nil {
		return
	}
	inst.node.LocalStreams = o.Streams

	capi, err := inst.node.IPFSCoreAPI()
	if err != nil {
		return err
	}
	inst.dsync = dsync.New(dag.NewNodeGetter(capi), capi.Block())

	return
}

// TODO (b5): this is a repo layout assertion, move to repo package?
func loadRepoConfig(repoPath string) (*config.Config, error) {
	path := filepath.Join(repoPath, "config.yaml")

	if _, e := os.Stat(path); os.IsNotExist(e) {
		return nil, nil
	}

	return config.ReadFromFile(path)
}

var (
	pluginLoadLock  sync.Once
	pluginLoadError error
)

func loadIPFSPluginsOnce(path string) error {
	body := func() {
		pluginLoadError = ipfs.LoadPlugins(path)
	}
	pluginLoadLock.Do(body)
	return pluginLoadError
}

func newStore(ctx context.Context, cfg *config.Config) (store cafs.Filestore, err error) {
	switch cfg.Store.Type {
	case "ipfs":
		path := cfg.Store.Path
		if path == "" && os.Getenv("IPFS_PATH") != "" {
			path = os.Getenv("IPFS_PATH")
		} else if path == "" {
			home, err := homedir.Dir()
			if err != nil {
				return nil, fmt.Errorf("creating IPFS store: %s", err)
			}
			path = filepath.Join(home, ".ipfs")
		}

		if err := loadIPFSPluginsOnce(path); err != nil {
			return nil, err
		}

		fsOpts := []ipfs.Option{
			func(c *ipfs.StoreCfg) {
				c.Ctx = ctx
				c.FsRepoPath = path
			},
			ipfs.OptsFromMap(cfg.Store.Options),
		}
		return ipfs.NewFilestore(fsOpts...)
	case "map":
		return cafs.NewMapstore(), nil
	default:
		return nil, fmt.Errorf("unknown store type: %s", cfg.Store.Type)
	}
}

func newRegClient(cfg *config.Config) (rc *regclient.Client) {
	if cfg.Registry != nil && cfg.Registry.Location != "" {
		rc = regclient.NewClient(&regclient.Config{
			Location: cfg.Registry.Location,
		})
	}
	return
}

func newRepo(path string, cfg *config.Config, store cafs.Filestore, rc *regclient.Client) (r repo.Repo, err error) {
	var pro *profile.Profile
	if pro, err = profile.NewProfile(cfg.Profile); err != nil {
		return
	}

	switch cfg.Repo.Type {
	case "fs":
		return fsrepo.NewRepo(store, nil, pro, rc, path)
	case "mem":
		return repo.NewMemRepo(pro, store, nil, profile.NewMemStore(), rc)
	default:
		return nil, fmt.Errorf("unknown repo type: %s", cfg.Repo.Type)
	}
}

func newFilesystem(cfg *config.Config, store cafs.Filestore) (qfs.Filesystem, error) {
	mux := map[string]qfs.PathResolver{
		"local": localfs.NewFS(),
		"http":  httpfs.NewFS(),
		"cafs":  store,
	}

	if ipfss, ok := store.(*ipfs.Filestore); ok {
		mux["ipfs"] = ipfss
	}

	fsys := muxfs.NewMux(mux)
	return fsys, nil
}

func newCron(cfg *config.Config, repoPath string) (cron.Scheduler, error) {
	updateCfg := cfg.Update
	if updateCfg == nil {
		updateCfg = config.DefaultUpdate()
	}

	cli := cron.HTTPClient{Addr: updateCfg.Address}
	if err := cli.Ping(); err == nil {
		return cli, nil
	}

	path, err := update.Path(repoPath)
	if err != nil {
		return nil, err
	}

	var jobStore, logStore cron.JobStore
	switch updateCfg.Type {
	case "fs":
		jobStore = cron.NewFlatbufferJobStore(filepath.Join(path, "jobs.qfb"))
		logStore = cron.NewFlatbufferJobStore(filepath.Join(path, "logs.qfb"))
	case "mem":
		jobStore = &cron.MemJobStore{}
		logStore = &cron.MemJobStore{}
	default:
		return nil, fmt.Errorf("unknown cron type: %s", updateCfg.Type)
	}

	svc := cron.NewCron(jobStore, logStore, update.Factory)
	return svc, nil
}

// NewInstanceFromConfigAndNode is a temporary solution to create an instance from an
// already-allocated QriNode & configuration
// don't write new code that relies on this, instead create a configuration
// and options that can be fed to NewInstance
func NewInstanceFromConfigAndNode(cfg *config.Config, node *p2p.QriNode) *Instance {
	ctx, teardown := context.WithCancel(context.Background())
	inst := &Instance{
		ctx:      ctx,
		teardown: teardown,
		cfg:      cfg,
		node:     node,
	}

	if node != nil && node.Repo != nil {
		inst.repo = node.Repo
		inst.store = node.Repo.Store()
		inst.qfs = node.Repo.Filesystem()
	}

	return inst
}

// Instance bundles the foundational values qri relies on, including a qri
// configuration, p2p node, and base context.
// An instance wraps required state for for "Method" constructors, which
// contain qri business logic. Think of instance as the "core" of the qri
// ecosystem. Create an Instance pointer with NewInstance
type Instance struct {
	ctx      context.Context
	teardown context.CancelFunc

	repoPath string
	cfg      *config.Config

	streams  ioes.IOStreams
	store    cafs.Filestore
	qfs      qfs.Filesystem
	registry *regclient.Client
	repo     repo.Repo
	node     *p2p.QriNode
	cron     cron.Scheduler
	dsync    *dsync.Dsync

	rpc *rpc.Client
}

// Context returns the base context for this instance
func (inst *Instance) Context() context.Context {
	return inst.ctx
}

// Config provides methods for manipulating Qri configuration
func (inst *Instance) Config() *config.Config {
	return inst.cfg
}

// RepoPath returns the path to the directory qri is operating from
func (inst *Instance) RepoPath() string {
	return inst.repoPath
}

// ChangeConfig implements the ConfigSetter interface
func (inst *Instance) ChangeConfig(cfg *config.Config) (err error) {

	if path := inst.cfg.Path(); path != "" {
		if err = cfg.WriteToFile(path); err != nil {
			return
		}
	}

	inst.cfg = cfg
	return nil
}

// Node accesses the instance qri node if one exists
func (inst *Instance) Node() *p2p.QriNode {
	return inst.node
}

// Repo accesses the instance Repo if one exists
func (inst *Instance) Repo() repo.Repo {
	if inst.repo != nil {
		return inst.repo
	} else if inst.node != nil {
		return inst.node.Repo
	}
	return nil
}

// RPC accesses the instance RPC client if one exists
func (inst *Instance) RPC() *rpc.Client {
	return inst.rpc
}

// Teardown destroys the instance, releasing reserved resources
func (inst *Instance) Teardown() {
	inst.teardown()
}
