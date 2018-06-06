package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
)

// peersCmd represents the info command
var peersCmd = &cobra.Command{
	Use:   "peers",
	Short: "commands for working with peers",
	Annotations: map[string]string{
		"group": "network",
	},
}

var peersInfoCmd = &cobra.Command{
	Use:   "info",
	Short: `Get info on a qri peer`,
	Long: `
The peers info command returns a peer's profile information. The default
format is yaml.`,
	Example: `  show info on a peer named "b5":
  $ qri peers info b5

  show info in json:
  $ qri peers info b5 --format json`,
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		req, err := peerRequests(false)
		ExitIfErr(err)

		v, err := cmd.Flags().GetBool("verbose")
		ExitIfErr(err)

		f, err := cmd.Flags().GetString("format")
		ExitIfErr(err)

		if !(f == "yaml" || f == "json") {
			ExitIfErr(fmt.Errorf("format must be either `yaml` or `json`"))
		}

		var data []byte

		p := &core.PeerInfoParams{
			Peername: args[0],
			Verbose:  v,
		}

		res := &config.ProfilePod{}
		err = req.Info(p, res)
		ExitIfErr(err)

		switch f {
		case "json":
			data, err = json.MarshalIndent(res, "", "  ")
			ExitIfErr(err)
		case "yaml":
			data, err = yaml.Marshal(res)
		}

		printInfo("\n" + string(data) + "\n")
	},
}

var peersListCmd = &cobra.Command{
	Use:   "list",
	Short: "list known qri peers",
	Long: `
lists the peers your qri node has seen before. The peers list command will
show the cached list of peers, unless you are currently running the connect
command in the background or in another terminal window.

(run 'qri help connect' for more information about the connect command) `,
	Example: `  to list qri peers:
  $ qri peers list

  to ensure you get a cached version of the list:
  $ qri peers list --cached`,
	Aliases: []string{"ls"},
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		ntwk, err := cmd.Flags().GetString("network")
		ExitIfErr(err)
		showCached, err := cmd.Flags().GetBool("cached")
		ExitIfErr(err)
		limit := 200

		// TODO - resurrect
		// outformat := cmd.Flag("format").Value.String()
		// if outformat != "" {
		// 	format, err := dataset.ParseDataFormatString(outformat)
		// 	if err != nil {
		// 		ErrExit(fmt.Errorf("invalid data format: %s", cmd.Flag("format").Value.String()))
		// 	}
		// 	if format != dataset.JSONDataFormat {
		// 		ErrExit(fmt.Errorf("invalid data format. currently only json or plaintext are supported"))
		// 	}
		// }

		req, err := peerRequests(false)
		ExitIfErr(err)

		if ntwk == "ipfs" {
			res := []string{}
			err := req.ConnectedIPFSPeers(&limit, &res)
			ExitIfErr(err)

			for i, p := range res {
				printSuccess("%d.\t%s", i+1, p)
			}
		} else {

			// if we don't have an RPC client, assume we're not connected
			if rpcConn() == nil && !showCached {
				printInfo("qri not connected, listing cached peers")
				showCached = true
			}

			p := &core.PeerListParams{
				Limit:  200,
				Offset: 0,
				Cached: showCached,
			}
			res := []*config.ProfilePod{}
			err = req.List(p, &res)
			ExitIfErr(err)

			fmt.Println("")
			for i, peer := range res {
				printPeerInfo(i, peer)
			}
		}

	},
}

var peersConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect to a peer",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		pr, err := peerRequests(false)
		ExitIfErr(err)

		pcpod := core.NewPeerConnectionParamsPod(args[0])
		res := &config.ProfilePod{}
		err = pr.ConnectToPeer(pcpod, res)
		ExitIfErr(err)

		printSuccess("successfully connected to %s:\n", res.Peername)
		printPeerInfo(0, res)
	},
}

var peersDisconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "explicitly close a connection to a peer",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		pr, err := peerRequests(false)
		ExitIfErr(err)

		pcpod := core.NewPeerConnectionParamsPod(args[0])
		res := false
		err = pr.DisconnectFromPeer(pcpod, &res)
		ExitIfErr(err)

		printSuccess("disconnected")
	},
}

func init() {
	peersInfoCmd.Flags().BoolP("verbose", "v", false, "show verbose profile info")
	peersInfoCmd.Flags().StringP("format", "", "yaml", "format in which to show peers profile info. formats: yaml, json")

	// peersListCmd.Flags().StringP("format", "f", "", "set output format [json]")
	peersListCmd.Flags().BoolP("cached", "c", false, "show peers that aren't online, but previously seen")

	peersCmd.AddCommand(peersInfoCmd, peersListCmd, peersConnectCmd, peersDisconnectCmd)
	RootCmd.AddCommand(peersCmd)
}
