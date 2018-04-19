package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/dataset"
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
	Example: `  show info on a user named "mr0grog":
  $ qri peers info mr0grog`,
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			ErrExit(fmt.Errorf("peer name is required"))
		}

		printInfo("searching for peer %s...", args[0])
		req, err := peerRequests(true)
		ExitIfErr(err)

		p := &core.PeerInfoParams{
			Peername: args[0],
		}

		res := &core.Profile{}
		err = req.Info(p, res)
		ExitIfErr(err)

		data, err := json.MarshalIndent(res, "", "  ")
		ExitIfErr(err)

		printSuccess(string(data))
	},
}

var peersListCmd = &cobra.Command{
	Use:   "list",
	Short: "list known qri peers",
	Long:  `lists the peers your qri node has seen before`,
	Example: `  list qri peers:
  $ qri peers list`,
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		outformat := cmd.Flag("format").Value.String()
		if outformat != "" {
			format, err := dataset.ParseDataFormatString(outformat)
			if err != nil {
				ErrExit(fmt.Errorf("invalid data format: %s", cmd.Flag("format").Value.String()))
			}
			if format != dataset.JSONDataFormat {
				ErrExit(fmt.Errorf("invalid data format. currently only json or plaintext are supported"))
			}
		}

		pr, err := peerRequests(false)
		ExitIfErr(err)

		res := []*core.Profile{}
		err = pr.List(&core.ListParams{Limit: 200}, &res)
		ExitIfErr(err)

		if len(res) == 0 {
			printWarning("no peers connected")
			return
		}

		if outformat == "" {
			for i, p := range res {
				printPeerInfo(i, p)
			}
		} else {
			data, err := json.MarshalIndent(res, "", "  ")
			ExitIfErr(err)
			fmt.Printf("%s", string(data))
		}
	},
}

var peersConnectCommand = &cobra.Command{
	Use:   "connect",
	Short: "connect directly to a peer ID",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		pr, err := peerRequests(false)
		ExitIfErr(err)

		res := &core.Profile{}
		err = pr.ConnectToPeer(&args[0], res)
		ExitIfErr(err)

		printPeerInfo(0, res)
	},
}

func init() {
	peersListCmd.Flags().StringP("format", "f", "", "set output format [json]")

	peersCmd.AddCommand(peersInfoCmd, peersListCmd, peersConnectCommand)
	RootCmd.AddCommand(peersCmd)
}
