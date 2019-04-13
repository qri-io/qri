package cmd

import (
	"fmt"
	"net/rpc"
	"sync"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo/gen"
	"github.com/spf13/cobra"
)

// NewQriCommand represents the base command when called without any subcommands
func NewQriCommand(pf PathFactory, generator gen.CryptoGenerator, ioStreams ioes.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qri",
		Short: "qri GDVCS CLI",
		Long: `
qri ("query") is a global dataset version control system 
on the distributed web.

https://qri.io

Feedback, questions, bug reports, and contributions are welcome!
https://github.com/qri-io/qri/issues`,
	}

	qriPath, ipfsPath := pf()
	opt := NewQriOptions(qriPath, ipfsPath, generator, ioStreams)

	// TODO: write a test that verifies this works with our new yaml config
	// RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $QRI_PATH/config.yaml)")
	cmd.SetUsageTemplate(rootUsageTemplate)
	cmd.PersistentFlags().BoolVarP(&opt.NoPrompt, "no-prompt", "", false, "disable all interactive prompts")
	cmd.PersistentFlags().BoolVarP(&opt.NoColor, "no-color", "", false, "disable colorized output")

	cmd.AddCommand(
		NewAddCommand(opt, ioStreams),
		NewConfigCommand(opt, ioStreams),
		NewConnectCommand(opt, ioStreams),
		NewDAGCommand(opt, ioStreams),
		NewDiffCommand(opt, ioStreams),
		NewExportCommand(opt, ioStreams),
		NewGetCommand(opt, ioStreams),
		NewListCommand(opt, ioStreams),
		NewLogCommand(opt, ioStreams),
		NewPublishCommand(opt, ioStreams),
		NewPeersCommand(opt, ioStreams),
		NewRegistryCommand(opt, ioStreams),
		NewRemoveCommand(opt, ioStreams),
		NewRenameCommand(opt, ioStreams),
		NewRenderCommand(opt, ioStreams),
		NewSaveCommand(opt, ioStreams),
		NewSearchCommand(opt, ioStreams),
		NewSetupCommand(opt, ioStreams),
		NewUseCommand(opt, ioStreams),
		NewUpdateCommand(opt, ioStreams),
		NewValidateCommand(opt, ioStreams),
		NewVersionCommand(opt, ioStreams),
	)

	for _, sub := range cmd.Commands() {
		sub.SetUsageTemplate(defaultUsageTemplate)
	}

	return cmd
}

// QriOptions holds the Root Command State
type QriOptions struct {
	// QriRepoPath is the path to the QRI repository
	qriRepoPath string
	// IpfsFsPath is the path to the IPFS repo
	ipfsFsPath string
	// NoPrompt Disables all promt messages
	NoPrompt bool
	// NoColor disables colorized output
	NoColor bool
	// path to configuration object
	ConfigPath string

	inst        lib.Instance
	initialized sync.Once
}

// NewQriOptions creates an options object
func NewQriOptions(qriPath, ipfsPath string, generator gen.CryptoGenerator, ioStreams ioes.IOStreams) *QriOptions {
	return &QriOptions{
		qriRepoPath: qriPath,
		ipfsFsPath:  ipfsPath,
		IOStreams:   ioStreams,
		generator:   generator,
	}
}

// Init will initialize the internal state
func (o *QriOptions) Init() (err error) {
	initBody := func() {
		o.inst, err = lib.NewInstance()
	}
	o.initialized.Do(initBody)
	return
	// initBody := func() {
	// 	cfgPath := filepath.Join(o.qriRepoPath, "config.yaml")

	// 	// TODO - need to remove global config state in lib, then remove this
	// 	lib.ConfigFilepath = cfgPath

	// 	if err = lib.LoadConfig(o.IOStreams, cfgPath); err != nil {
	// 		return
	// 	}
	// 	o.config = lib.Config

	// 	setNoColor(!o.config.CLI.ColorizeOutput || o.NoColor)

	// 	if o.config.RPC.Enabled {
	// 		addr := fmt.Sprintf(":%d", o.config.RPC.Port)
	// 		if conn, err := net.Dial("tcp", addr); err != nil {
	// 			err = nil
	// 		} else {
	// 			o.rpc = rpc.NewClient(conn)
	// 			return
	// 		}
	// 	}

	// 	// for now this just checks for an existing config file
	// 	if _, e := os.Stat(cfgPath); os.IsNotExist(e) {
	// 		err = fmt.Errorf("no qri repo found, please run `qri setup`")
	// 		return
	// 	}

	// 	var store *ipfs.Filestore

	// 	fsOpts := []ipfs.Option{
	// 		func(cfg *ipfs.StoreCfg) {
	// 			cfg.FsRepoPath = o.ipfsFsPath
	// 			// cfg.Online = online
	// 		},
	// 		ipfs.OptsFromMap(o.config.Store.Options),
	// 	}

	// 	store, err = ipfs.NewFilestore(fsOpts...)
	// 	if err != nil {
	// 		return
	// 	}

	// 	var pro *profile.Profile
	// 	if pro, err = profile.NewProfile(o.config.Profile); err != nil {
	// 		return
	// 	}

	// 	var rc *regclient.Client
	// 	if o.config.Registry != nil && o.config.Registry.Location != "" {
	// 		rc = regclient.NewClient(&regclient.Config{
	// 			Location: o.config.Registry.Location,
	// 		})
	// 	}

	// 	fsys := muxfs.NewMux(map[string]qfs.PathResolver{
	// 		"local": localfs.NewFS(),
	// 		"http":  httpfs.NewFS(),
	// 		"cafs":  store,
	// 		"ipfs":  store,
	// 	})

	// 	o.repo, err = fsrepo.NewRepo(store, fsys, pro, rc, o.qriRepoPath)
	// 	if err != nil {
	// 		return
	// 	}

	// 	o.node, err = p2p.NewQriNode(o.repo, o.config.P2P)
	// 	if err != nil {
	// 		return
	// 	}
	// 	o.node.LocalStreams = o.IOStreams
	// }
	// o.initialized.Do(initBody)
	// return err
}

// Instance returns the instance this options is using
func (o *QriOptions) Instance() lib.Instance {
	if err := o.Init(); err != nil {
		return nil
	}
	return o.inst
}

// Config returns from internal state
func (o *QriOptions) Config() (*config.Config, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return o.inst.Config(), nil
}

// IpfsFsPath returns from internal state
func (o *QriOptions) IpfsFsPath() string {
	return o.ipfsFsPath
}

// QriRepoPath returns from internal state
func (o *QriOptions) QriRepoPath() string {
	return o.qriRepoPath
}

// CryptoGenerator returns a resource for generating cryptographic info
func (o *QriOptions) CryptoGenerator() gen.CryptoGenerator {
	return o.generator
}

// RPC returns from internal state
func (o *QriOptions) RPC() *rpc.Client {
	if err := o.Init(); err != nil {
		return nil
	}
	return o.inst.RPC()
}

// ConnectionNode returns the internal QriNode, if it is available
func (o *QriOptions) ConnectionNode() (*p2p.QriNode, error) {
	if o.inst == nil {
		return nil, fmt.Errorf("repo not available")
	}
	return o.inst.Node(), nil
}

// DatasetRequests generates a lib.DatasetRequests from internal state
func (o *QriOptions) DatasetRequests() (*lib.DatasetRequests, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewDatasetRequests(o.inst.Node(), o.inst.RPC()), nil
}

// RemoteRequests generates a lib.RemoteRequests from internal state
func (o *QriOptions) RemoteRequests() (*lib.RemoteRequests, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewRemoteRequests(o.inst.Node(), o.inst.Config(), o.inst.RPC()), nil
}

// RegistryRequests generates a lib.RegistryRequests from internal state
func (o *QriOptions) RegistryRequests() (*lib.RegistryRequests, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewRegistryRequests(o.inst.Node(), o.inst.RPC()), nil
}

// LogRequests generates a lib.LogRequests from internal state
func (o *QriOptions) LogRequests() (*lib.LogRequests, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewLogRequests(o.inst.Node(), o.inst.RPC()), nil
}

// ExportRequests generates a lib.ExportRequests from internal state
func (o *QriOptions) ExportRequests() (*lib.ExportRequests, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewExportRequests(o.inst.Node(), o.inst.RPC()), nil
}

// PeerRequests generates a lib.PeerRequests from internal state
func (o *QriOptions) PeerRequests() (*lib.PeerRequests, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewPeerRequests(nil, o.inst.RPC()), nil
}

// ProfileRequests generates a lib.ProfileRequests from internal state
func (o *QriOptions) ProfileRequests() (*lib.ProfileRequests, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	setCfg := func(*config.Config) error { return nil }
	if writable, ok := o.inst.(lib.WritableInstance); ok {
		setCfg = writable.SetConfig
	}
	return lib.NewProfileRequests(o.inst.Node(), o.inst.Config(), setCfg, o.inst.RPC()), nil
}

// SelectionRequests creates a lib.SelectionRequests from internal state
func (o *QriOptions) SelectionRequests() (*lib.SelectionRequests, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewSelectionRequests(o.inst.Repo(), o.inst.RPC()), nil
}

// SearchRequests generates a lib.SearchRequests from internal state
func (o *QriOptions) SearchRequests() (*lib.SearchRequests, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewSearchRequests(o.inst.Node(), o.inst.RPC()), nil
}

// RenderRequests generates a lib.RenderRequests from internal state
func (o *QriOptions) RenderRequests() (*lib.RenderRequests, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewRenderRequests(o.inst.Repo(), o.inst.RPC()), nil
}

// ConfigRequests generates a lib.ConfigRequests from internal state
func (o *QriOptions) ConfigRequests() (*lib.ConfigRequests, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	var setCfg func(*config.Config) error
	if writable, ok := o.inst.(lib.WritableInstance); ok {
		setCfg = writable.SetConfig
	}
	return lib.NewConfigRequests(o.inst.Config(), setCfg, o.inst.RPC()), nil
}
