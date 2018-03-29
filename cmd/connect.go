package cmd

import (
	"github.com/qri-io/qri/api"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	connectCmdPort       string
	connectCmdRPCPort    string
	connectCmdWebappPort string

	disableP2P      bool
	connectSetup    bool
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
peers & swapping data.

The default port for the local API server is 2503. We call port 2503,
“the qri port”. It’s a good port, lots of cool numbers in there. Some might even
call it a “prime” port number.`,
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
			*c = *cfg

			if connectCmdPort != config.DefaultAPIPort {
				c.API.Enabled = connectCmdPort != ""
				c.API.Port = connectCmdPort
			}

			if connectCmdRPCPort != config.DefaultRPCPort {
				c.RPC.Enabled = connectCmdPort != ""
				c.RPC.Port = connectCmdRPCPort
			}

			if connectCmdWebappPort != config.DefaultWebappPort {
				c.Webapp.Enabled = connectCmdWebappPort != ""
				c.RPC.Port = connectCmdWebappPort
			}
		})

		ExitIfErr(err)

		err = s.Serve()
		ExitIfErr(err)
	},
}

func init() {
	connectCmd.Flags().StringVarP(&connectCmdPort, "api-port", "", config.DefaultAPIPort, "port to start api on")
	connectCmd.Flags().StringVarP(&connectCmdRPCPort, "rpc-port", "", config.DefaultRPCPort, "port to start rpc listener on")
	connectCmd.Flags().StringVarP(&connectCmdWebappPort, "webapp-port", "", config.DefaultWebappPort, "port to serve webapp on")
	connectCmd.Flags().BoolVarP(&connectSetup, "setup", "", false, "run setup if necessary, reading options from enviornment variables")
	connectCmd.Flags().BoolVarP(&disableP2P, "disable-p2p", "", false, "disable peer-2-peer networking")
	connectCmd.Flags().BoolVarP(&connectReadOnly, "read-only", "", false, "run qri in read-only mode, limits the api endpoints")
	RootCmd.AddCommand(connectCmd)
}
