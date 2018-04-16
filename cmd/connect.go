package cmd

import (
	"fmt"
	"github.com/qri-io/qri/api"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	connectCmdAPIPort         int
	connectCmdRPCPort         int
	connectCmdWebappPort      int
	connectCmdDisconnectAfter int

	disableAPI    bool
	disableRPC    bool
	disableWebapp bool

	connectSetup       bool
	connectCmdRegistry string
	disableP2P         bool
	connectReadOnly    bool
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
		if !connectSetup {
			loadConfig()
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			r   repo.Repo
			err error
		)

		if connectSetup && !QRIRepoInitialized() {
			err = doSetup("", "", connectCmdRegistry, false)
			ExitIfErr(err)
		} else if !QRIRepoInitialized() {
			ErrExit(fmt.Errorf("no qri repo exists"))
		}

		r = getRepo(true)

		s, err := api.New(r, func(c *config.Config) {
			*c = *core.Config

			if connectCmdAPIPort != 0 {
				c.API.Port = connectCmdAPIPort
			}

			if connectCmdRPCPort != 0 {
				c.RPC.Port = connectCmdRPCPort
			}

			if connectCmdWebappPort != 0 {
				c.Webapp.Port = connectCmdWebappPort
			}

			if connectCmdDisconnectAfter != 0 {
				c.API.DisconnectAfter = connectCmdDisconnectAfter
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
		if err != nil && err.Error() == "http: Server closed" {
			return
		}
		ExitIfErr(err)
	},
}

func init() {
	connectCmd.Flags().IntVarP(&connectCmdAPIPort, "api-port", "", 0, "port to start api on")
	connectCmd.Flags().IntVarP(&connectCmdRPCPort, "rpc-port", "", 0, "port to start rpc listener on")
	connectCmd.Flags().IntVarP(&connectCmdWebappPort, "webapp-port", "", 0, "port to serve webapp on")
	connectCmd.Flags().IntVarP(&connectCmdDisconnectAfter, "disconnect-after", "", 0, "duration to keep connected in seconds, 0 means run indefinitely")

	connectCmd.Flags().BoolVarP(&disableAPI, "disable-api", "", false, "disables api, overrides the api-port flag")
	connectCmd.Flags().BoolVarP(&disableRPC, "disable-rpc", "", false, "disables rpc, overrides the rpc-port flag")
	connectCmd.Flags().BoolVarP(&disableWebapp, "disable-webapp", "", false, "disables webapp, overrides the webapp-port flag")
	connectCmd.Flags().BoolVarP(&disableP2P, "disable-p2p", "", false, "disable peer-2-peer networking")

	connectCmd.Flags().BoolVarP(&connectSetup, "setup", "", false, "run setup if necessary, reading options from enviornment variables")
	connectCmd.Flags().BoolVarP(&connectReadOnly, "read-only", "", false, "run qri in read-only mode, limits the api endpoints")
	connectCmd.Flags().StringVarP(&connectCmdRegistry, "registry", "", "", "specify registry to setup with. only works when --setup is true")
	RootCmd.AddCommand(connectCmd)
}
