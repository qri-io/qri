package cmd

import (
	"context"
	"fmt"
	"net/rpc"
	"os"
	"sync"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo/gen"
	"github.com/spf13/cobra"
)

// NewQriCommand represents the base command when called without any subcommands
func NewQriCommand(ctx context.Context, pf PathFactory, generator gen.CryptoGenerator, ioStreams ioes.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qri",
		Short: "qri GDVCS CLI",
		Long: `qri ("query") is a global dataset version control system 
on the distributed web.

https://qri.io

Feedback, questions, bug reports, and contributions are welcome!
https://github.com/qri-io/qri/issues`,
	}

	qriPath, ipfsPath := pf()
	opt := NewQriOptions(ctx, qriPath, ipfsPath, generator, ioStreams)

	cmd.SetUsageTemplate(rootUsageTemplate)
	cmd.PersistentFlags().BoolVarP(&opt.NoPrompt, "no-prompt", "", false, "disable all interactive prompts")
	cmd.PersistentFlags().BoolVarP(&opt.NoColor, "no-color", "", false, "disable colorized output")
	cmd.PersistentFlags().StringVar(&opt.RepoPath, "repo", qriPath, "provide a path to load qri from")
	cmd.PersistentFlags().StringVar(&opt.IpfsPath, "ipfs-path", ipfsPath, "override IPFS path location")
	cmd.PersistentFlags().BoolVarP(&opt.LogAll, "log-all", "", false, "log all activity")

	cmd.AddCommand(
		NewAddCommand(opt, ioStreams),
		NewCheckoutCommand(opt, ioStreams),
		NewConfigCommand(opt, ioStreams),
		NewConnectCommand(opt, ioStreams),
		NewDAGCommand(opt, ioStreams),
		NewDiffCommand(opt, ioStreams),
		NewExportCommand(opt, ioStreams),
		NewFetchCommand(opt, ioStreams),
		NewFSICommand(opt, ioStreams),
		NewGetCommand(opt, ioStreams),
		NewInitCommand(opt, ioStreams),
		NewListCommand(opt, ioStreams),
		NewLogCommand(opt, ioStreams),
		NewLogbookCommand(opt, ioStreams),
		NewPublishCommand(opt, ioStreams),
		NewPeersCommand(opt, ioStreams),
		NewRegistryCommand(opt, ioStreams),
		NewRemoveCommand(opt, ioStreams),
		NewRenameCommand(opt, ioStreams),
		NewRenderCommand(opt, ioStreams),
		NewRestoreCommand(opt, ioStreams),
		NewSaveCommand(opt, ioStreams),
		NewSearchCommand(opt, ioStreams),
		NewSetupCommand(opt, ioStreams),
		NewStatsCommand(opt, ioStreams),
		NewStatusCommand(opt, ioStreams),
		NewSQLCommand(opt, ioStreams),
		NewUseCommand(opt, ioStreams),
		NewUpdateCommand(opt, ioStreams),
		NewValidateCommand(opt, ioStreams),
		NewVersionCommand(opt, ioStreams),
		NewWhatChangedCommand(opt, ioStreams),
	)

	for _, sub := range cmd.Commands() {
		sub.SetUsageTemplate(defaultUsageTemplate)
	}

	return cmd
}

// QriOptions holds the Root Command State
type QriOptions struct {
	ioes.IOStreams

	// TODO (b5) - this context should be refactored away, prefering to pass
	// this stored context object down through function calls
	ctx context.Context

	// path to the QRI repository directory
	RepoPath string
	// custom path to IPFS repo
	IpfsPath string
	// generator is source of generating cryptographic info
	generator gen.CryptoGenerator
	// NoPrompt Disables all promt messages
	NoPrompt bool
	// NoColor disables colorized output
	NoColor bool
	// path to configuration object
	ConfigPath string
	// Whether to log all activity by enabling logging for all packages
	LogAll bool

	inst        *lib.Instance
	initialized sync.Once
}

// NewQriOptions creates an options object
func NewQriOptions(ctx context.Context, qriPath, ipfsPath string, generator gen.CryptoGenerator, ioStreams ioes.IOStreams) *QriOptions {
	return &QriOptions{
		IOStreams: ioStreams,
		ctx:       ctx,
		RepoPath:  qriPath,
		IpfsPath:  ipfsPath,
		generator: generator,
	}
}

// Init will initialize the internal state
func (o *QriOptions) Init() (err error) {
	initBody := func() {
		opts := []lib.Option{
			lib.OptIOStreams(o.IOStreams), // transfer iostreams to instance
			lib.OptSetIPFSPath(o.IpfsPath),
			lib.OptCheckConfigMigrations(""),
			lib.OptSetLogAll(o.LogAll),
		}
		o.inst, err = lib.NewInstance(o.ctx, o.RepoPath, opts...)
		log.Debugf("running cmd %q", os.Args)
	}
	o.initialized.Do(initBody)
	return
}

// Instance returns the instance this options is using
func (o *QriOptions) Instance() *lib.Instance {
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
	return o.IpfsPath
}

// QriRepoPath returns from internal state
func (o *QriOptions) QriRepoPath() string {
	return o.RepoPath
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
	return lib.NewDatasetRequestsInstance(o.inst), nil
}

// RemoteMethods generates a lib.RemoteMethods from internal state
func (o *QriOptions) RemoteMethods() (*lib.RemoteMethods, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewRemoteMethods(o.inst), nil
}

// RegistryClientMethods generates a lib.RegistryClientMethods from internal state
func (o *QriOptions) RegistryClientMethods() (*lib.RegistryClientMethods, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewRegistryClientMethods(o.inst), nil
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
func (o *QriOptions) ProfileMethods() (m *lib.ProfileMethods, err error) {
	if err = o.Init(); err != nil {
		return
	}

	return lib.NewProfileMethods(o.inst), nil
}

// SearchMethods generates a lib.SearchMethods from internal state
func (o *QriOptions) SearchMethods() (*lib.SearchMethods, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewSearchMethods(o.inst), nil
}

// SQLMethods generates a lib.SQLMethods from internal state
func (o *QriOptions) SQLMethods() (*lib.SQLMethods, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewSQLMethods(o.inst), nil
}

// RenderRequests generates a lib.RenderRequests from internal state
func (o *QriOptions) RenderRequests() (*lib.RenderRequests, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewRenderRequests(o.inst.Repo(), o.inst.RPC()), nil
}

// ConfigMethods generates a lib.ConfigMethods from internal state
func (o *QriOptions) ConfigMethods() (m *lib.ConfigMethods, err error) {
	if err = o.Init(); err != nil {
		return
	}

	return lib.NewConfigMethods(o.inst), nil
}

// FSIMethods generates a lib.FSIMethods from internal state
func (o *QriOptions) FSIMethods() (m *lib.FSIMethods, err error) {
	if err = o.Init(); err != nil {
		return
	}

	return lib.NewFSIMethods(o.inst), nil
}
