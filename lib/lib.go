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
	"github.com/mitchellh/go-homedir"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
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

// Instance is the interface that bundles the foundational values of a
// qri instance.
type Instance interface {
	// Context returns the base context this instance is using. Any resources
	// built from this instance should inherit from this context and obey
	// calls from the ctx.Done(), releasing any & all resources
	Context() context.Context
	// the returned context should tear down the entire instance & cleanup all
	// resources the instance is currently consuming
	// Teardown closes the instance by closing the base context
	Teardown()
	// Config returns the current configuration for this
	Config() *config.Config
	Node() *p2p.QriNode
	Repo() repo.Repo
	RPC() *rpc.Client
}

// WritableInstance is an instance that can be modified
// instances that allow config changes should implement this interface
type WritableInstance interface {
	// SetConfig modifies configuration details
	SetConfig(*config.Config) error
}

// ErrNotWritable is a canonical error for try to edit an instance
// that isn't editable
var ErrNotWritable = fmt.Errorf("this instance isn't writable")

// Methods is a related set of library functions
type Methods interface {
	// MethodsKind confirms participation in the Methods interface while
	// also giving a human readable string for logging purposes
	MethodsKind() string
}

// AllMethods returns a slice of CoreRequests that defines the full local
// API of lib methods
func AllMethods(inst Instance) []Methods {
	return []Methods{
		// NewDatasetRequests(node, nil),
		// NewRegistryRequests(node, nil),
		// NewLogRequests(node, nil),
		// NewExportRequests(node, nil),
		// NewPeerRequests(node, nil),
		// NewProfileRequests(node, cfg, cfgFilepath, nil),
		// NewSearchRequests(node, nil),
		// NewRenderRequests(node.Repo, nil),
		// NewSelectionRequests(node.Repo, nil),
	}
}

// InstanceOptions provides details to NewInstance.
// New will alter InstanceOptions by applying
// any provided Option functions
// to distinguish "Options" from "Config":
// * Options contains state that can only be determined at runtime
// * Config consists only of static values stored in a configuration file
// Options may override config in specific cases to avoid undefined state
type InstanceOptions struct {
	Cfg      *config.Config
	Streams  ioes.IOStreams
	QriPath  string
	IPFSPath string
}

// Option is a function that manipulates config details when fed to New()
type Option func(o *InstanceOptions) error

// OptDefaultQriPath configures the directory to read Qri from, defaulting to
// "$HOME/.qri", unless the environment variable QRI_PATH is set
func OptDefaultQriPath() Option {
	return func(o *InstanceOptions) error {
		path := os.Getenv("QRI_PATH")
		if path == "" {
			dir, err := homedir.Dir()
			if err != nil {
				return err
			}
			path = filepath.Join(dir, ".qri")
		}
		o.QriPath = path
		return nil
	}
}

// OptDefaultIPFSPath configures the directory to read IPFS from, defaulting to
// "$HOME/.ipfs", unless the environment variable IPFS_PATH is set
func OptDefaultIPFSPath() Option {
	return func(o *InstanceOptions) error {
		path := os.Getenv("IPFS_PATH")
		if path == "" {
			dir, err := homedir.Dir()
			if err != nil {
				return err
			}
			path = filepath.Join(dir, ".ipfs")
		}
		o.IPFSPath = path
		return nil
	}
}

// OptLoadConfigFile loads a configuration from a given path
func OptLoadConfigFile(path string) Option {
	return func(o *InstanceOptions) (err error) {
		// default to checking
		if path == "" && o.QriPath != "" {
			path = filepath.Join(o.QriPath, "config.yaml")
		} else if path == "" {
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
		if cfgPath == "" && o.QriPath != "" {
			cfgPath = filepath.Join(o.QriPath, "config.yaml")
		} else if cfgPath == "" {
			return fmt.Errorf("no config path provided")
		}

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
			OptStdIOStreams(),
			OptDefaultQriPath(),
			OptDefaultIPFSPath(),
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

	ctx, teardown := context.WithCancel(context.Background())

	inst := &instance{
		ctx:      ctx,
		teardown: teardown,
		cfg:      cfg,
	}
	qri = inst

	// configure logging straight away
	if cfg != nil && cfg.Logging != nil {
		for name, level := range cfg.Logging.Levels {
			golog.SetLogLevel(name, level)
		}
	}

	if cfg.RPC.Enabled {
		addr := fmt.Sprintf(":%d", cfg.RPC.Port)
		if conn, err := net.Dial("tcp", addr); err != nil {
			err = nil
		} else {
			inst.rpc = rpc.NewClient(conn)
			return qri, err
		}
	}

	var store *ipfs.Filestore
	fsOpts := []ipfs.Option{
		func(c *ipfs.StoreCfg) {
			c.FsRepoPath = o.IPFSPath
			// c.Online = online
		},
		ipfs.OptsFromMap(cfg.Store.Options),
	}
	if store, err = ipfs.NewFilestore(fsOpts...); err != nil {
		return
	}

	var pro *profile.Profile
	if pro, err = profile.NewProfile(cfg.Profile); err != nil {
		return
	}

	var rc *regclient.Client
	if cfg.Registry != nil && cfg.Registry.Location != "" {
		rc = regclient.NewClient(&regclient.Config{
			Location: cfg.Registry.Location,
		})
	}

	fsys := muxfs.NewMux(map[string]qfs.PathResolver{
		"local": localfs.NewFS(),
		"http":  httpfs.NewFS(),
		"cafs":  store,
		"ipfs":  store,
	})

	repo, err := fsrepo.NewRepo(store, fsys, pro, rc, o.QriPath)
	if err != nil {
		return
	}

	node, err := p2p.NewQriNode(repo, cfg.P2P)
	if err != nil {
		return
	}
	node.LocalStreams = o.Streams

	inst.node = node
	inst.qriPath = o.QriPath

	return
}

// instance implements the (exported) Instance interface
// create an instance one with NewInstance
type instance struct {
	ctx      context.Context
	teardown context.CancelFunc

	cfg     *config.Config
	qriPath string
	node    *p2p.QriNode
	rpc     *rpc.Client
}

// Context returns the base context for this instance
func (inst *instance) Context() context.Context {
	return inst.ctx
}

// Config provides methods for manipulating Qri configuration
func (inst *instance) Config() *config.Config {
	return inst.cfg
}

// SetConfig implements the ConfigSetter interface
func (inst *instance) SetConfig(cfg *config.Config) error {
	if inst.qriPath != "" {
		// TODO (b5): write to config file here
	}
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
