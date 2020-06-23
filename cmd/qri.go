package cmd

import (
	"context"
	"fmt"
	"net/rpc"
	"os"
	"runtime"
	"sync"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo/gen"
	"github.com/spf13/cobra"
)

// NewQriCommand represents the base command when called without any subcommands
func NewQriCommand(ctx context.Context, repoPath string, generator gen.CryptoGenerator, ioStreams ioes.IOStreams) (*cobra.Command, func() <-chan error) {
	opt := NewQriOptions(ctx, repoPath, generator, ioStreams)

	cmd := &cobra.Command{
		Use:   "qri",
		Short: "qri GDVCS CLI",
		Long: `qri ("query") is a set of tools for building & sharing datasets: https://qri.io
Feedback, questions, bug reports, and contributions are welcome! 
https://github.com/qri-io/qri/issues`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
		},
		BashCompletionFunction: bashCompletionFunc,
	}

	cmd.SetUsageTemplate(rootUsageTemplate)
	cmd.PersistentFlags().BoolVarP(&opt.NoPrompt, "no-prompt", "", false, "disable all interactive prompts")
	cmd.PersistentFlags().BoolVarP(&opt.NoColor, "no-color", "", false, "disable colorized output")
	cmd.PersistentFlags().StringVar(&opt.repoPath, "repo", repoPath, "filepath to load qri data from")
	cmd.PersistentFlags().BoolVarP(&opt.LogAll, "log-all", "", false, "log all activity")

	cmd.AddCommand(
		NewAddCommand(opt, ioStreams),
		NewAutocompleteCommand(opt, ioStreams),
		NewCheckoutCommand(opt, ioStreams),
		NewConfigCommand(opt, ioStreams),
		NewConnectCommand(opt, ioStreams),
		NewDAGCommand(opt, ioStreams),
		NewDiffCommand(opt, ioStreams),
		NewExportCommand(opt, ioStreams),
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
		NewValidateCommand(opt, ioStreams),
		NewVersionCommand(opt, ioStreams),
		NewWhatChangedCommand(opt, ioStreams),
	)

	for _, sub := range cmd.Commands() {
		sub.SetUsageTemplate(defaultUsageTemplate)
	}

	return cmd, opt.Shutdown
}

// QriOptions holds the Root Command State
type QriOptions struct {
	ioes.IOStreams

	// TODO (b5) - this context should be refactored away, prefering to pass
	// this stored context object down through function calls
	ctx       context.Context
	releasers sync.WaitGroup
	doneCh    chan struct{}

	// path to the qri data directory
	repoPath string
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
	// inst is the Instance that holds state needed by qri's methods
	inst *lib.Instance
}

// NewQriOptions creates an options object
func NewQriOptions(ctx context.Context, repoPath string, generator gen.CryptoGenerator, ioStreams ioes.IOStreams) *QriOptions {
	return &QriOptions{
		IOStreams: ioStreams,
		ctx:       ctx,
		doneCh:    make(chan struct{}),
		repoPath:  repoPath,
		generator: generator,
	}
}

// Init will initialize the internal state
func (o *QriOptions) Init() (err error) {
	if o.inst != nil {
		return
	}
	opts := []lib.Option{
		lib.OptIOStreams(o.IOStreams), // transfer iostreams to instance
		lib.OptCheckConfigMigrations(!noPrompt),
		lib.OptSetLogAll(o.LogAll),
	}
	o.inst, err = lib.NewInstance(o.ctx, o.repoPath, opts...)
	if err != nil {
		return
	}

	// Handle color and prompt flags which apply to every command
	shouldColorOutput := !o.NoColor
	cfg := o.inst.Config()
	if cfg != nil && cfg.CLI != nil {
		shouldColorOutput = cfg.CLI.ColorizeOutput
	}
	// todo(arqu): have a config var to indicate force override for windows
	if runtime.GOOS == "windows" {
		shouldColorOutput = false
	}
	setNoColor(!shouldColorOutput)
	setNoPrompt(o.NoPrompt)
	log.Debugf("running cmd %q", os.Args)

	return
}

// Instance returns the instance this options is using
func (o *QriOptions) Instance() *lib.Instance {
	if err := o.Init(); err != nil {
		return nil
	}
	return o.inst
}

// RepoPath returns the path to the qri data directory
func (o *QriOptions) RepoPath() string {
	return o.repoPath
}

// Config returns from internal state
func (o *QriOptions) Config() (*config.Config, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return o.inst.Config(), nil
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

// DatasetMethods generates a lib.DatasetMethods from internal state
func (o *QriOptions) DatasetMethods() (*lib.DatasetMethods, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewDatasetMethods(o.inst), nil
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

// LogMethods generates a lib.LogMethods from internal state
func (o *QriOptions) LogMethods() (*lib.LogMethods, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewLogMethods(o.inst), nil
}

// ExportRequests generates a lib.ExportRequests from internal state
func (o *QriOptions) ExportRequests() (*lib.ExportRequests, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewExportRequests(o.inst.Node(), o.inst.RPC()), nil
}

// PeerMethods generates a lib.PeerMethods from internal state
func (o *QriOptions) PeerMethods() (*lib.PeerMethods, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewPeerMethods(o.inst), nil
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

// RenderMethods generates a lib.RenderMethods from internal state
func (o *QriOptions) RenderMethods() (*lib.RenderMethods, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return lib.NewRenderMethods(o.inst), nil
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

// Shutdown closes the instance
func (o *QriOptions) Shutdown() <-chan error {
	if o.inst == nil {
		done := make(chan error)
		go func() { done <- nil }()
		return done
	}
	return o.inst.Shutdown()
}
