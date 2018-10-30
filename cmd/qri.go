package cmd

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"sync"

	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/fs"
	"github.com/qri-io/qri/repo/gen"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry/regclient"
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
	cmd.Flags().BoolVarP(&opt.NoPrompt, "no-prompt", "", false, "disable all interactive prompts")
	cmd.Flags().BoolVarP(&opt.NoColor, "no-color", "", false, "disable colorized output")

	cmd.AddCommand(
		NewAddCommand(opt, ioStreams),
		NewConfigCommand(opt, ioStreams),
		NewConnectCommand(opt, ioStreams),
		NewBodyCommand(opt, ioStreams),
		NewDiffCommand(opt, ioStreams),
		NewExportCommand(opt, ioStreams),
		NewGetCommand(opt, ioStreams),
		NewInfoCommand(opt, ioStreams),
		NewListCommand(opt, ioStreams),
		NewLogCommand(opt, ioStreams),
		NewPublishCommand(opt, ioStreams),
		NewPeersCommand(opt, ioStreams),
		// TODO - registry command is currently removed in favor of the newer "qri publish" command
		// we should consider refactoring this code (espcially it's documentation) &
		// use it for registry-specific publication & search interaction
		// NewRegistryCommand(opt, ioStreams),
		NewRemoveCommand(opt, ioStreams),
		NewRenameCommand(opt, ioStreams),
		NewRenderCommand(opt, ioStreams),
		NewSaveCommand(opt, ioStreams),
		NewSearchCommand(opt, ioStreams),
		NewSetupCommand(opt, ioStreams),
		NewUseCommand(opt, ioStreams),
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

	// Configuration object
	config      *config.Config
	node        *p2p.QriNode
	repo        repo.Repo
	rpc         *rpc.Client
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

func (o *QriOptions) init() (err error) {
	initBody := func() {
		cfgPath := filepath.Join(o.qriRepoPath, "config.yaml")

		// TODO - need to remove global config state in lib, then remove this
		lib.ConfigFilepath = cfgPath

		if err = lib.LoadConfig(cfgPath); err != nil {
			return
		}
		o.config = lib.Config

		setNoColor(!o.config.CLI.ColorizeOutput || o.NoColor)

		if o.config.RPC.Enabled {
			addr := fmt.Sprintf(":%d", o.config.RPC.Port)
			if conn, err := net.Dial("tcp", addr); err != nil {
				err = nil
			} else {
				o.rpc = rpc.NewClient(conn)
				return
			}
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

		o.node, err = p2p.NewQriNode(o.repo, o.config.P2P)
		if err != nil {
			return
		}
		o.node.LocalStreams = o.IOStreams
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

// CryptoGenerator returns a resource for generating cryptographic info
func (o *QriOptions) CryptoGenerator() gen.CryptoGenerator {
	return o.generator
}

// RPC returns from internal state
func (o *QriOptions) RPC() *rpc.Client {
	return o.rpc
}

// Repo returns from internal state
func (o *QriOptions) Repo() (repo.Repo, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	if o.repo == nil {
		return nil, fmt.Errorf("repo not available (are you running qri in another terminal?)")
	}
	return o.repo, nil
}

// Node returns the internal QriNode
func (o *QriOptions) Node() (*p2p.QriNode, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	if o.repo == nil {
		return nil, fmt.Errorf("repo not available (are you running qri in another terminal?)")
	}
	return o.node, nil
}

// DatasetRequests generates a lib.DatasetRequests from internal state
func (o *QriOptions) DatasetRequests() (*lib.DatasetRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return lib.NewDatasetRequests(o.node, o.rpc), nil
}

// RegistryRequests generates a lib.RegistryRequests from internal state
func (o *QriOptions) RegistryRequests() (*lib.RegistryRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return lib.NewRegistryRequests(o.node, o.rpc), nil
}

// LogRequests generates a lib.LogRequests from internal state
func (o *QriOptions) LogRequests() (*lib.LogRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return lib.NewLogRequests(o.node, o.rpc), nil
}

// PeerRequests generates a lib.PeerRequests from internal state
func (o *QriOptions) PeerRequests() (*lib.PeerRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return lib.NewPeerRequests(nil, o.rpc), nil
}

// ProfileRequests generates a lib.ProfileRequests from internal state
func (o *QriOptions) ProfileRequests() (*lib.ProfileRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return lib.NewProfileRequests(o.node, o.rpc), nil
}

// SelectionRequests creates a lib.SelectionRequests from internal state
func (o *QriOptions) SelectionRequests() (*lib.SelectionRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return lib.NewSelectionRequests(o.repo, o.rpc), nil
}

// SearchRequests generates a lib.SearchRequests from internal state
func (o *QriOptions) SearchRequests() (*lib.SearchRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return lib.NewSearchRequests(o.node, o.rpc), nil
}

// RenderRequests generates a lib.RenderRequests from internal state
func (o *QriOptions) RenderRequests() (*lib.RenderRequests, error) {
	if err := o.init(); err != nil {
		return nil, err
	}
	return lib.NewRenderRequests(o.repo, o.rpc), nil
}
