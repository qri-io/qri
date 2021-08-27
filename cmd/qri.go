package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	qhttp "github.com/qri-io/qri/lib/http"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/remote/access"
	"github.com/spf13/cobra"
)

// NewQriCommand represents the base command when called without any subcommands
func NewQriCommand(ctx context.Context, repoPath string, generator key.CryptoGenerator, ioStreams ioes.IOStreams, libOpts ...lib.Option) (*cobra.Command, func() <-chan error) {
	opt := NewQriOptions(ctx, repoPath, generator, ioStreams, libOpts)

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
	cmd.PersistentFlags().BoolVarP(&opt.Migrate, "migrate", "", false, "automatically run migrations if necessary")
	cmd.PersistentFlags().BoolVarP(&opt.NoPrompt, "no-prompt", "", false, "disable all interactive prompts")
	cmd.PersistentFlags().BoolVarP(&opt.NoColor, "no-color", "", false, "disable colorized output")
	cmd.PersistentFlags().StringVar(&opt.repoPath, "repo", repoPath, "filepath to load qri data from")
	cmd.PersistentFlags().BoolVarP(&opt.LogAll, "log-all", "", false, "log all activity")

	cmd.AddCommand(
		NewAccessCommand(opt, ioStreams),
		NewApplyCommand(opt, ioStreams),
		NewAutocompleteCommand(opt, ioStreams),
		NewConfigCommand(opt, ioStreams),
		NewConnectCommand(opt, ioStreams),
		NewDAGCommand(opt, ioStreams),
		NewDiffCommand(opt, ioStreams),
		NewGetCommand(opt, ioStreams),
		NewListCommand(opt, ioStreams),
		NewLogCommand(opt, ioStreams),
		NewLogbookCommand(opt, ioStreams),
		NewPushCommand(opt, ioStreams),
		NewPullCommand(opt, ioStreams),
		NewPeersCommand(opt, ioStreams),
		NewPreviewCommand(opt, ioStreams),
		NewRegistryCommand(opt, ioStreams),
		NewRemoveCommand(opt, ioStreams),
		NewRenameCommand(opt, ioStreams),
		NewRenderCommand(opt, ioStreams),
		NewSaveCommand(opt, ioStreams),
		NewSearchCommand(opt, ioStreams),
		NewSetupCommand(opt, ioStreams),
		NewValidateCommand(opt, ioStreams),
		NewVersionCommand(opt, ioStreams),
	)

	for _, sub := range cmd.Commands() {
		sub.SetUsageTemplate(defaultUsageTemplate)
	}

	return cmd, opt.Shutdown
}

// QriOptions holds the Root Command State
type QriOptions struct {
	ioes.IOStreams

	// TODO (b5) - this context should be refactored away, preferring to pass
	// this stored context object down through function calls
	ctx       context.Context
	releasers sync.WaitGroup
	doneCh    chan struct{}

	// path to the qri data directory
	repoPath string
	// generator is source of generating cryptographic info
	generator key.CryptoGenerator
	// automatically run migrations if necessary
	Migrate bool
	// NoPrompt Disables all promt messages
	NoPrompt bool
	// NoColor disables colorized output
	NoColor bool
	// path to configuration object
	ConfigPath string
	// Whether to log all activity by enabling logging for all packages
	LogAll  bool
	libOpts []lib.Option
	// inst is the Instance that holds state needed by qri's methods
	inst *lib.Instance
}

// NewQriOptions creates an options object
func NewQriOptions(ctx context.Context, repoPath string, generator key.CryptoGenerator, ioStreams ioes.IOStreams, libOpts []lib.Option) *QriOptions {
	return &QriOptions{
		IOStreams: ioStreams,
		ctx:       ctx,
		doneCh:    make(chan struct{}),
		repoPath:  repoPath,
		libOpts:   libOpts,
		generator: generator,
	}
}

// Init will initialize the internal state before any command is run (excluding `qri setup`)
func (o *QriOptions) Init() (err error) {
	if o.inst != nil {
		return
	}
	setNoPrompt(o.NoPrompt)

	repoErr := lib.QriRepoExists(o.repoPath)
	if repoErr != nil {
		return errors.New("no qri repo exists\nhave you run 'qri setup'?")
	}

	opts := []lib.Option{
		lib.OptIOStreams(o.IOStreams), // transfer iostreams to instance
		lib.OptCheckConfigMigrations(o.migrationApproval, (!o.Migrate && !o.NoPrompt)),
		lib.OptSetLogAll(o.LogAll),
		lib.OptRemoteServerOptions([]remote.OptionsFunc{
			// look for a remote policy
			remote.OptLoadPolicyFileIfExists(filepath.Join(o.repoPath, access.DefaultAccessControlPolicyFilename)),
		}),
	}

	if o.libOpts != nil {
		opts = append(o.libOpts, o.libOpts...)
	}

	o.inst, err = lib.NewInstance(o.ctx, o.repoPath, opts...)
	if err != nil {
		return
	}

	// Handle color and prompt flags which apply to every command
	shouldColorOutput := !o.NoColor
	cfg := o.inst.GetConfig()
	if cfg != nil && cfg.CLI != nil {
		shouldColorOutput = cfg.CLI.ColorizeOutput
	}
	// todo(arqu): have a config var to indicate force override for windows
	if runtime.GOOS == "windows" {
		shouldColorOutput = false
	}
	setNoColor(!shouldColorOutput)

	// TODO (b5) - this is a hack to make progress bars not show up while running
	// tests. It does have the real-world implication that "shouldColorOutput"
	// being false also disables progress bars, which may be what we want (ahem: TTY
	// detection), but even if so, isn't the right use of this variable name
	if shouldColorOutput {
		// TODO(ramfox): we guard for a nil bus in `PrintProgressBarsOnEvents`
		// but noting here that no requests that go through http rpc will have
		// a working bus, so we won't get any progress bars when working over
		// http rpc until this is adjusted (once we get the events "rpc-ified")
		PrintProgressBarsOnEvents(o.IOStreams.ErrOut, o.inst.Bus())
	}

	log.Debugf("running cmd %q", os.Args)

	return
}

// migrationApproval returns a boolen based on either flag-derived state or user
// input approving the execution of migrations
func (o *QriOptions) migrationApproval() bool {
	if o.Migrate {
		return true
	} else if o.NoPrompt {
		return false
	}

	msg := `Your repo needs updating before qri can start. 
Run migration now?`
	return confirm(o.Out, o.In, msg, false)
}

// Instance returns the instance this options is using
func (o *QriOptions) Instance() (*lib.Instance, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	return o.inst, nil
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
	return o.inst.GetConfig(), nil
}

// CryptoGenerator returns a resource for generating cryptographic info
func (o *QriOptions) CryptoGenerator() key.CryptoGenerator {
	return o.generator
}

// HTTPClient returns a client for performing RPC over HTTP
func (o *QriOptions) HTTPClient() *qhttp.Client {
	if err := o.Init(); err != nil {
		return nil
	}
	return o.inst.HTTPClient()
}

// ConnectionNode returns the internal QriNode, if it is available
func (o *QriOptions) ConnectionNode() (*p2p.QriNode, error) {
	if o.inst == nil {
		return nil, fmt.Errorf("repo not available")
	}
	return o.inst.Node(), nil
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
