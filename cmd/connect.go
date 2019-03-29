package cmd

import (
	"fmt"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/api"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/spf13/cobra"
)

// NewConnectCommand creates a new `qri connect` cobra command for connecting to the d.web, local api, rpc server, and webapp
func NewConnectCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := ConnectOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to the distributed web by spinning up a Qri node",
		Annotations: map[string]string{
			"group": "network",
		},
		Long: `
While it’s not totally accurate, connect is like starting a server. Running 
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
			return o.Run()
		},
	}

	cmd.Flags().IntVarP(&o.APIPort, "api-port", "", 0, "port to start api on")
	cmd.Flags().IntVarP(&o.RPCPort, "rpc-port", "", 0, "port to start rpc listener on")
	cmd.Flags().IntVarP(&o.WebappPort, "webapp-port", "", 0, "port to serve webapp on")
	cmd.Flags().IntVarP(&o.DisconnectAfter, "disconnect-after", "", 0, "duration to keep connected in seconds, 0 means run indefinitely")

	cmd.Flags().BoolVarP(&o.DisableAPI, "disable-api", "", false, "disables api, overrides the api-port flag")
	cmd.Flags().BoolVarP(&o.DisableRPC, "disable-rpc", "", false, "disables rpc, overrides the rpc-port flag")
	cmd.Flags().BoolVarP(&o.DisableWebapp, "disable-webapp", "", false, "disables webapp, overrides the webapp-port flag")
	cmd.Flags().BoolVarP(&o.DisableP2P, "disable-p2p", "", false, "disables webapp, overrides the webapp-port flag")
	// TODO - not yet supported
	// cmd.Flags().BoolVarP(&o.DisableP2P, "disable-p2p", "", false, "disable peer-2-peer networking")

	cmd.Flags().BoolVarP(&o.Setup, "setup", "", false, "run setup if necessary, reading options from environment variables")
	cmd.Flags().BoolVarP(&o.ReadOnly, "read-only", "", false, "run qri in read-only mode, limits the api endpoints")
	cmd.Flags().BoolVarP(&o.RemoteMode, "remote-mode", "", false, "run qri in remote mode")
	cmd.Flags().Int64VarP(&o.RemoteAcceptSizeMax, "remote-accept-size-max", "", -1, "when running as a remote, max size of dataset to accept, -1 for any size")
	cmd.Flags().StringVarP(&o.Registry, "registry", "", "", "specify registry to setup with. only works when --setup is true")

	return cmd
}

// ConnectOptions encapsulates state for the connect command
type ConnectOptions struct {
	ioes.IOStreams

	APIPort         int
	RPCPort         int
	WebappPort      int
	DisconnectAfter int

	DisableAPI    bool
	DisableRPC    bool
	DisableWebapp bool
	DisableP2P    bool

	Registry            string
	Setup               bool
	ReadOnly            bool
	RemoteMode          bool
	RemoteAcceptSizeMax int64

	Node   *p2p.QriNode
	Config *config.Config
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ConnectOptions) Complete(f Factory, args []string) (err error) {
	qriPath := f.QriRepoPath()

	if o.Setup && !QRIRepoInitialized(qriPath) {
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
	} else if !QRIRepoInitialized(qriPath) {
		return fmt.Errorf("no qri repo exists")
	}

	if err = f.Init(); err != nil {
		return err
	}
	o.Node, err = f.ConnectionNode()
	if err != nil {
		return fmt.Errorf("%s, is `qri connect` already running?", err)
	}
	o.Config, err = f.Config()
	return
}

// Run executes the connect command with currently configured state
func (o *ConnectOptions) Run() (err error) {
	cfg := *o.Config

	if o.APIPort != 0 {
		cfg.API.Port = o.APIPort
	}
	if o.RPCPort != 0 {
		cfg.RPC.Port = o.RPCPort
	}
	if o.WebappPort != 0 {
		cfg.Webapp.Port = o.WebappPort
	}

	if o.DisconnectAfter != 0 {
		cfg.API.DisconnectAfter = o.DisconnectAfter
	}

	if o.ReadOnly {
		cfg.API.ReadOnly = true
	}
	if o.RemoteMode {
		cfg.API.RemoteMode = true
	}
	if o.DisableP2P {
		cfg.P2P.Enabled = false
	}
	if o.DisableAPI {
		cfg.API.Enabled = false
	}
	if o.DisableRPC {
		cfg.RPC.Enabled = false
	}
	if o.DisableWebapp {
		cfg.Webapp.Enabled = false
	}

	cfg.API.RemoteAcceptSizeMax = o.RemoteAcceptSizeMax

	s := api.New(o.Node, &cfg)
	err = s.Serve()
	if err != nil && err.Error() == "http: Server closed" {
		return nil
	}
	return err
}
