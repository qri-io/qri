package cmd

import (
	"context"
	"fmt"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/api"
	"github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewConnectCommand creates a new `qri connect` cobra command for connecting to the d.web, local api, and rpc server
func NewConnectCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := ConnectOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "connect to the distributed web by spinning up a Qri node",
		Annotations: map[string]string{
			"group": "network",
		},
		Long: `While it’s not totally accurate, connect is like starting a server. Running 
connect will start a process and stay there until you exit the process 
(ctrl+c from the terminal, or killing the process using tools like activity 
monitor on the mac, or the aptly-named “kill” command). Connect does three main 
things:
- Connect to the qri distributed network
- Connect to IPFS
- Start a local API server

When you run connect you are connecting to the distributed web, interacting with
peers & swapping data.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().BoolVarP(&o.Setup, "setup", "", false, "run setup if necessary, reading options from environment variables")
	cmd.Flags().StringVarP(&o.Registry, "registry", "", "", "specify registry to setup with. only works when --setup is true")

	return cmd
}

// ConnectOptions encapsulates state for the connect command
type ConnectOptions struct {
	ioes.IOStreams
	inst     *lib.Instance
	Registry string
	Setup    bool
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ConnectOptions) Complete(f Factory, args []string) (err error) {
	repoPath := f.RepoPath()

	if o.Setup && !QRIRepoInitialized(repoPath) {
		so := &SetupOptions{
			IOStreams: o.IOStreams,
			IPFS:      true,
			Registry:  o.Registry,
			Anonymous: true,
		}
		if err = so.Complete(f, args); err != nil {
			return err
		}
		if err = so.DoSetup(f); err != nil {
			return err
		}
	} else if !QRIRepoInitialized(repoPath) {
		return errors.New(repo.ErrNoRepo, "no qri repo exists\nhave you run 'qri setup'?")
	}

	if err = f.Init(); err != nil {
		return err
	}

	// This fails whenever `qri connect` runs but another instance of `qri connect` is already
	// running. If early in the connection process, this call to ConnectionNode will return an
	// error. If later in the process, ConnectionNode will return without error but also with
	// no node allocated. Without this check, later code will fail or segfault, might as well
	// fail early.
	n, err := f.ConnectionNode()
	if err != nil {
		return fmt.Errorf("%s, is `qri connect` already running?", err)
	}
	if n == nil {
		return fmt.Errorf("Cannot serve without a node (`qri connect` or Qri Desktop already running?)")
	}

	o.inst = f.Instance()
	return
}

// Run executes the connect command with currently configured state
func (o *ConnectOptions) Run() error {
	ctx := context.Background()
	err := api.New(o.inst).Serve(ctx)
	if err != nil && err.Error() == "http: Server closed" {
		return nil
	}
	return err
}
