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

	golog "github.com/ipfs/go-log"
	homedir "github.com/mitchellh/go-homedir"
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
	"github.com/qri-io/qri/repo/fs"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry/regclient"
)

var log = golog.Logger("lib")

// VersionNumber is the current version qri
const VersionNumber = "0.7.4-dev"

func init() {
	// Fields like dataset.Structure.Schema contain data of arbitrary types,
	// registering with the gob package prevents errors when sending them
	// over net/rpc calls.
	gob.Register([]interface{}{})
	gob.Register(map[string]interface{}{})
}

// Receivers returns a slice of CoreRequests that defines the full local
// API of lib methods
func Receivers(inst Instance) []Methods {
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
		NewSelectionRequests(r, nil),
	}
}

// Instance is the interface that bundles the foundational values of a
// qri instance. Instance provides the basis for creating Method constructors,
// which actually do qri things
// think of instance as the "core" of the qri ecosystem
type Instance interface {
	// Context returns the base context this instance is using. Any resources
	// built from this instance should inherit from this context and obey
	// calls from the ctx.Done(), releasing any & all resources
	Context() context.Context
	// Teardown closes the instance by closing the base context
	Teardown()
	// Config returns the current configuration for this
	Config() *config.Config
	// ChangeConfig modifies the configuration details of this instance
	// TODO (b5): changes to configuration need to update the runtime Instance
	// and derived methods
	ChangeConfig(*config.Config) error
	Node() *p2p.QriNode
	Repo() repo.Repo
	RPC() *rpc.Client
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
	Ctx     context.Context
	Cfg     *config.Config
	Streams ioes.IOStreams
}

// Option is a function that manipulates config details when fed to New()
type Option func(o *InstanceOptions) error

// OptBackgroundCtx uses a default base context
func OptBackgroundCtx() Option {
	return func(o *InstanceOptions) error {
		o.Ctx = context.Background()
		return nil
	}
}

// OptSetQriRepoPath configures the directory to read Qri from, defaulting to
// "$HOME/.qri", unless the environment variable QRI_PATH is set
func OptSetQriRepoPath(path string) Option {
	return func(o *InstanceOptions) error {
		if o.Cfg.Repo.Type == "fs" && o.Cfg.Repo.Path == "" {
			if path == "" {
				path := os.Getenv("QRI_PATH")
				if path == "" {
					dir, err := homedir.Dir()
					if err != nil {
						return err
					}
					path = filepath.Join(dir, ".qri")
				}
			}
			o.Cfg.Repo.Path = path
		}
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
		if o.Cfg.Store.Type == "ipfs" && o.Cfg.Store.Path == "" {
			if path == "" {
				path = os.Getenv("IPFS_PATH")
				if path == "" {
					dir, err := homedir.Dir()
					if err != nil {
						return err
					}
					path = filepath.Join(dir, ".ipfs")
				}
			}
			o.Cfg.Store.Path = path
		}
		return nil
	}
}

// OptLoadConfigFile loads a configuration from a given path
func OptLoadConfigFile(path string) Option {
	return func(o *InstanceOptions) (err error) {
		// default to checking
		// if path == "" {
		// 	path = filepath.Join(o.QriPath, "config.yaml")
		// } else if path == "" {
		// return fmt.Errorf("no config path provided")
		// }

		if path == "" {
			return fmt.Errorf("no config path provided")
		}

		if _, e := os.Stat(path); os.IsNotExist(e) {
			return fmt.Errorf("no qri repo found, please run `qri setup`")
		}

		o.Cfg, err = config.ReadFromFile(path)
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
		// default to checking
		// if cfgPath == "" && o.QriPath != "" {
		// 	cfgPath = filepath.Join(o.QriPath, "config.yaml")
		// } else if cfgPath == "" {
		// 	return fmt.Errorf("no config path provided")
		// }

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
// New uses a default set of Option funcs
func NewInstance(opts ...Option) (qri Instance, err error) {
	o := &InstanceOptions{}
	if len(opts) == 0 {
		// default to a standard composition of Option funcs
		opts = []Option{
			OptBackgroundCtx(),
			OptStdIOStreams(),
			OptSetQriRepoPath(""),
			OptSetIPFSPath(""),
			OptLoadConfigFile(""),
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
		err = fmt.Errorf("no configuration details provided")
		return
	}
	if err = cfg.Validate(); err != nil {
		return
	}

	ctx, teardown := context.WithCancel(o.Ctx)
	inst := &instance{
		ctx:      ctx,
		teardown: teardown,
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

	// check if we're operating over RPC
	if cfg.RPC.Enabled {
		addr := fmt.Sprintf(":%d", cfg.RPC.Port)
		if conn, err := net.Dial("tcp", addr); err != nil {
			err = nil
		} else {
			inst.rpc = rpc.NewClient(conn)
			return qri, err
		}
	}

	if inst.store, err = newStore(cfg); err != nil {
		return
	}
	if inst.qfs, err = newFilesystem(cfg, inst.store); err != nil {
		return
	}
	inst.registry = newRegClient(cfg)

	if inst.repo, err = newRepo(cfg, inst.store, inst.registry); err != nil {
		return
	}
	if qfssetter, ok := inst.repo.(repo.QFSSetter); ok {
		qfssetter.SetFilesystem(inst.qfs)
	}

	if inst.node, err = p2p.NewQriNode(inst.repo, cfg.P2P); err != nil {
		return
	}
	inst.node.LocalStreams = o.Streams

	return
}

func newStore(cfg *config.Config) (store cafs.Filestore, err error) {
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

		fsOpts := []ipfs.Option{
			func(c *ipfs.StoreCfg) {
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

func newRepo(cfg *config.Config, store cafs.Filestore, rc *regclient.Client) (r repo.Repo, err error) {
	var pro *profile.Profile
	if pro, err = profile.NewProfile(cfg.Profile); err != nil {
		return
	}

	switch cfg.Repo.Type {
	case "fs":
		return fsrepo.NewRepo(store, nil, pro, rc, cfg.Repo.Path)
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

// NewInstanceFromConfigAndNode is a temporary solution to create an instance from an
// already-allocated QriNode & configuration
// don't write new code that relies on this, instead create a configuration
// and options that can be fed to NewInstance
func NewInstanceFromConfigAndNode(cfg *config.Config, node *p2p.QriNode) Instance {
	ctx, teardown := context.WithCancel(context.Background())
	return &instance{
		ctx:      ctx,
		teardown: teardown,
		cfg:      cfg,
		node:     node,
	}
}

// instance implements the (exported) Instance interface
// create an instance one with NewInstance
type instance struct {
	ctx      context.Context
	teardown context.CancelFunc

	cfg *config.Config

	streams  ioes.IOStreams
	store    cafs.Filestore
	qfs      qfs.Filesystem
	registry *regclient.Client
	repo     repo.Repo
	node     *p2p.QriNode

	rpc *rpc.Client
}

// Context returns the base context for this instance
func (inst *instance) Context() context.Context {
	return inst.ctx
}

// Config provides methods for manipulating Qri configuration
func (inst *instance) Config() *config.Config {
	return inst.cfg
}

// ChangeConfig implements the ConfigSetter interface
func (inst *instance) ChangeConfig(cfg *config.Config) (err error) {

	if path := inst.cfg.Path(); path != "" {
		if err = cfg.WriteToFile(path); err != nil {
			return
		}
	}

	inst.cfg = cfg
	return nil
}

// Node accesses the instance qri node if one exists
func (inst *instance) Node() *p2p.QriNode {
	return inst.node
}

// Repo accesses the instance Repo if one exists
func (inst *instance) Repo() repo.Repo {
	if inst.node == nil {
		return nil
	}
	return inst.node.Repo
}

// RPC accesses the instance RPC client if one exists
func (inst *instance) RPC() *rpc.Client {
	return inst.rpc
}

// Teardown destroys the instance, releasing reserved resources
func (inst *instance) Teardown() {
	inst.teardown()
}
