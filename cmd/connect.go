package cmd

import (
	"github.com/qri-io/qri/api"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	connectCmdAPIPort    string
	connectCmdRPCPort    string
	connectCmdWebappPort string

	disableAPI    bool
	disableRPC    bool
	disableWebapp bool

	connectSetup    bool
	disableP2P      bool
	connectReadOnly bool
)

// connectCmd represents the run command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect to the distributed web, start a local API server",
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
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			r   repo.Repo
			err error
		)

		if connectSetup && !QRIRepoInitialized() {
			setupCmd.Run(&cobra.Command{}, []string{})
		}

		r = getRepo(true)

		s, err := api.New(r, func(c *config.Config) {
			*c = *core.Config

			if connectCmdAPIPort != "" {
				c.API.Port = connectCmdAPIPort
			}

			if connectCmdRPCPort != "" {
				c.RPC.Port = connectCmdRPCPort
			}

			if connectCmdWebappPort != "" {
				c.Webapp.Port = connectCmdWebappPort
			}

			if connectReadOnly {
				c.API.ReadOnly = true
			}

			if disableP2P {
				c.P2P.Enabled = false
			}

			if disableAPI {
				c.API.Enabled = false
			}

			if disableRPC {
				c.RPC.Enabled = false
			}

			if disableWebapp {
				c.Webapp.Enabled = false
			}
		})
		ExitIfErr(err)

		err = s.Serve()
		ExitIfErr(err)
	},
}

func init() {
	connectCmd.Flags().StringVarP(&connectCmdAPIPort, "api-port", "", "", "port to start api on")
	connectCmd.Flags().StringVarP(&connectCmdRPCPort, "rpc-port", "", "", "port to start rpc listener on")
	connectCmd.Flags().StringVarP(&connectCmdWebappPort, "webapp-port", "", "", "port to serve webapp on")

	connectCmd.Flags().BoolVarP(&disableAPI, "disable-api", "", false, "disables api, overrides the api-port flag")
	connectCmd.Flags().BoolVarP(&disableRPC, "disable-rpc", "", false, "disables rpc, overrides the rpc-port flag")
	connectCmd.Flags().BoolVarP(&disableWebapp, "disable-webapp", "", false, "disables webapp, overrides the webapp-port flag")
	connectCmd.Flags().BoolVarP(&disableP2P, "disable-p2p", "", false, "disable peer-2-peer networking")

	connectCmd.Flags().BoolVarP(&connectSetup, "setup", "", false, "run setup if necessary, reading options from enviornment variables")
	connectCmd.Flags().BoolVarP(&connectReadOnly, "read-only", "", false, "run qri in read-only mode, limits the api endpoints")
	RootCmd.AddCommand(connectCmd)
}
