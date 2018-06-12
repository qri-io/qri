package cmd

import (
	"fmt"
	"io"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"sync"

	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/fs"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry/regclient"
	"github.com/spf13/cobra"
)

// NewQriCommand represents the base command when called without any subcommands
func NewQriCommand(pf PathFactory, in io.Reader, out, err io.Writer) *cobra.Command {
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

	ioStreams := IOStreams{In: in, Out: out, ErrOut: err}
	qriPath, ipfsPath := pf()
	opt := NewQriOptions(qriPath, ipfsPath, ioStreams)

	// TODO: write a test that verifies this works with our new yaml config
	// RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $QRI_PATH/config.yaml)")
	cmd.SetUsageTemplate(rootUsageTemplate)
	cmd.Flags().BoolVarP(&opt.NoPrompt, "no-prompt", "", false, "disable all interactive prompts")
	cmd.Flags().BoolVarP(&opt.NoColor, "no-color", "", false, "disable colorized output")

	cmd.AddCommand(
		NewAddCommand(opt, ioStreams),
		NewConfigCommand(opt, ioStreams),
		NewConnectCommand(opt, ioStreams),
		NewDataCommand(opt, ioStreams),
		NewDiffCommand(opt, ioStreams),
		NewExportCommand(opt, ioStreams),
		NewInfoCommand(opt, ioStreams),
		NewListCommand(opt, ioStreams),
		NewLogCommand(opt, ioStreams),
		NewPeersCommand(opt, ioStreams),
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

	return cmd
}

// QriOptions holds the Root Command State
type QriOptions struct {
	IOStreams
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

	// Configuration object
	config      *config.Config
	node        *p2p.QriNode
	repo        repo.Repo
	rpc         *rpc.Client
	initialized sync.Once
}

// NewQriOptions creates an options object
func NewQriOptions(qriPath, ipfsPath string, ioStreams IOStreams) *QriOptions {
	return &QriOptions{
		qriRepoPath: qriPath,
		ipfsFsPath:  ipfsPath,
		IOStreams:   ioStreams,
	}
}

func (o *QriOptions) init() (err error) {
	initBody := func() {
		cfgPath := filepath.Join(o.qriRepoPath, "config.yaml")

		// TODO - need to remove global config state in core, then remove this
		core.ConfigFilepath = cfgPath

		if err = core.LoadConfig(cfgPath); err != nil {
			return
		}
		o.config = core.Config
		setNoColor(o.config.CLI.ColorizeOutput || o.NoColor)

		addr := fmt.Sprintf(":%d", o.config.RPC.Port)
		if conn, err := net.Dial("tcp", addr); err != nil {
			err = nil
		} else {
			o.rpc = rpc.NewClient(conn)
			return
		}

		// for now this just checks for an existing config file
		if _, e := os.Stat(cfgPath); os.IsNotExist(e) {
			err = fmt.Errorf("no qri repo found, please run `qri setup`")
			return
		}

		var fs *ipfs.Filestore
		fs, err = ipfs.NewFilestore(func(cfg *ipfs.StoreCfg) {
			cfg.FsRepoPath = o.ipfsFsPath
			// cfg.Online = online
		})
		if err != nil {
			return
		}

		var pro *profile.Profile
		if pro, err = profile.NewProfile(o.config.Profile); err != nil {
			return
		}

		var rc *regclient.Client
		if o.config.Registry != nil && o.config.Registry.Location != "" {
			rc = regclient.NewClient(&regclient.Config{
				Location: o.config.Registry.Location,
			})
		}

		o.repo, err = fsrepo.NewRepo(fs, pro, rc, o.qriRepoPath)
		if err != nil {
			return
		}

	}
	o.initialized.Do(initBody)
	return err
}

// Config returns from internal state
func (o *QriOptions) Config() (*config.Config, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return o.config, nil
}

// IpfsFsPath returns from internal state
func (o *QriOptions) IpfsFsPath() string {
	return o.ipfsFsPath
}

// QriRepoPath returns from internal state
func (o *QriOptions) QriRepoPath() string {
	return o.qriRepoPath
}

// RPC returns from internal state
func (o *QriOptions) RPC() *rpc.Client {
	return o.rpc
}

// Repo returns from internal state
func (o *QriOptions) Repo() (repo.Repo, error) {
	return o.repo, o.init()
}

// DatasetRequests generates a core.DatasetRequests from internal state
func (o *QriOptions) DatasetRequests() (*core.DatasetRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return core.NewDatasetRequests(o.repo, o.rpc), nil
}

// RegistryRequests generates a core.RegistryRequests from internal state
func (o *QriOptions) RegistryRequests() (*core.RegistryRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return core.NewRegistryRequests(o.repo, o.rpc), nil
}

// HistoryRequests generates a core.HistoryRequests from internal state
func (o *QriOptions) HistoryRequests() (*core.HistoryRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return core.NewHistoryRequests(o.repo, o.rpc), nil
}

// PeerRequests generates a core.PeerRequests from internal state
func (o *QriOptions) PeerRequests() (*core.PeerRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return core.NewPeerRequests(nil, o.rpc), nil
}

// ProfileRequests generates a core.ProfileRequests from internal state
func (o *QriOptions) ProfileRequests() (*core.ProfileRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return core.NewProfileRequests(o.repo, o.rpc), nil
}

// SearchRequests generates a core.SearchRequests from internal state
func (o *QriOptions) SearchRequests() (*core.SearchRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return core.NewSearchRequests(o.repo, o.rpc), nil
}

// RenderRequests generates a core.RenderRequests from internal state
func (o *QriOptions) RenderRequests() (*core.RenderRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return core.NewRenderRequests(o.repo, o.rpc), nil
}
