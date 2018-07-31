package cmd

import (
	"fmt"

	"github.com/qri-io/qri/api"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewConnectCommand creates a new `qri connect` cobra command for connecting to the d.web, local api, rpc server, and webapp
func NewConnectCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := ConnectOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "connect to the distributed web, start a local API server",
		Annotations: map[string]string{
			"group": "network",
		},
		Long: `
While it’s not totally accurate, connect is like starting a server. running 
connect will start a process and stay there until you exit the process 
(ctrl+c from the terminal, or killing the process using tools like activity 
monitor on the mac, or the aptly-named “kill” command). Connect does three main 
things:
- Connect to the qri distributed network
- Connect to IPFS
- Start a local API server

When you run connect you are connecting to the distributed web, interacting with
peers & swapping data.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().IntVarP(&o.APIPort, "api-port", "", 0, "port to start api on")
	cmd.Flags().IntVarP(&o.RPCPort, "rpc-port", "", 0, "port to start rpc listener on")
	cmd.Flags().IntVarP(&o.WebappPort, "webapp-port", "", 0, "port to serve webapp on")
	cmd.Flags().IntVarP(&o.DisconnectAfter, "disconnect-after", "", 0, "duration to keep connected in seconds, 0 means run indefinitely")

	cmd.Flags().BoolVarP(&o.DisableAPI, "disable-api", "", false, "disables api, overrides the api-port flag")
	cmd.Flags().BoolVarP(&o.DisableRPC, "disable-rpc", "", false, "disables rpc, overrides the rpc-port flag")
	cmd.Flags().BoolVarP(&o.DisableWebapp, "disable-webapp", "", false, "disables webapp, overrides the webapp-port flag")
	// TODO - not yet supported
	// cmd.Flags().BoolVarP(&o.DisableP2P, "disable-p2p", "", false, "disable peer-2-peer networking")

	cmd.Flags().BoolVarP(&o.Setup, "setup", "", false, "run setup if necessary, reading options from enviornment variables")
	cmd.Flags().BoolVarP(&o.ReadOnly, "read-only", "", false, "run qri in read-only mode, limits the api endpoints")
	cmd.Flags().StringVarP(&o.Registry, "registry", "", "", "specify registry to setup with. only works when --setup is true")

	return cmd
}

// ConnectOptions encapsulates state for the connect command
type ConnectOptions struct {
	IOStreams

	APIPort         int
	RPCPort         int
	WebappPort      int
	DisconnectAfter int

	DisableAPI    bool
	DisableRPC    bool
	DisableWebapp bool
	DisableP2P    bool

	Registry string
	Setup    bool
	ReadOnly bool

	Repo   repo.Repo
	Config *config.Config
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ConnectOptions) Complete(f Factory, args []string) (err error) {
	qriPath := f.QriRepoPath()

	if o.Setup && !QRIRepoInitialized(qriPath) {
		so := &SetupOptions{IOStreams: o.IOStreams, IPFS: true, Registry: o.Registry}
		if err = so.Complete(f, args); err != nil {
			return err
		}
		if err = so.DoSetup(f); err != nil {
			return err
		}
	} else if !QRIRepoInitialized(qriPath) {
		return fmt.Errorf("no qri repo exists")
	}

	// TODO - calling f.Repo has the side effect of
	// calling init if we haven't initialized so far. Should this be made
	// more explicit?
	o.Repo, err = f.Repo()
	if err != nil {
		return err
	}
	o.Config, err = f.Config()
	return
}

// Run executes the connect command with currently configured state
func (o *ConnectOptions) Run() (err error) {
	s, err := api.New(o.Repo, func(c *config.Config) {
		*c = *o.Config

		if o.APIPort != 0 {
			c.API.Port = o.APIPort
		}
		if o.RPCPort != 0 {
			c.RPC.Port = o.RPCPort
		}
		if o.WebappPort != 0 {
			c.Webapp.Port = o.WebappPort
		}

		if o.DisconnectAfter != 0 {
			c.API.DisconnectAfter = o.DisconnectAfter
		}

		if o.ReadOnly {
			c.API.ReadOnly = true
		}
		if o.DisableP2P {
			c.P2P.Enabled = false
		}
		if o.DisableAPI {
			c.API.Enabled = false
		}
		if o.DisableRPC {
			c.RPC.Enabled = false
		}
		if o.DisableWebapp {
			c.Webapp.Enabled = false
		}
	})
	if err != nil {
		return err
	}

	err = s.Serve()
	if err != nil && err.Error() == "http: Server closed" {
		return nil
	}
	return err
}
