package cmd

import (
	"fmt"
	"net/rpc"
	"path/filepath"
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
	ioes.IOStreams

	// QriRepoPath is the path to the QRI repository
	qriRepoPath string
	// IpfsFsPath is the path to the IPFS repo
	ipfsFsPath string
	// generator is source of generating cryptographic info
	generator gen.CryptoGenerator
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
		cfgPath := filepath.Join(o.qriRepoPath, "config.yaml")
		opts := []lib.Option{
			lib.OptLoadConfigFile(cfgPath),
			lib.OptBackgroundCtx(),        // build from a new context
			lib.OptIOStreams(o.IOStreams), // transfer iostreams down
			lib.OptSetQriRepoPath(o.qriRepoPath),
			lib.OptSetIPFSPath(o.ipfsFsPath),
			lib.OptCheckConfigMigrations(""),
		}
		o.inst, err = lib.NewInstance(opts...)
	}
	o.initialized.Do(initBody)
	return
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

// ProfileMethods generates a lib.ProfileMethods from internal state
func (o *QriOptions) ProfileMethods() (m lib.ProfileMethods, err error) {
	if err = o.Init(); err != nil {
		return
	}

	return lib.NewProfileMethods(o.inst), nil
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

// ConfigMethods generates a lib.ConfigMethods from internal state
func (o *QriOptions) ConfigMethods() (m lib.ConfigMethods, err error) {
	if err = o.Init(); err != nil {
		return
	}

	return lib.NewConfigMethods(o.inst), nil
}
