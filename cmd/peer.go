package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo/profile"
	"github.com/spf13/cobra"
)

// peerCmd represents the peer command
var peerCmd = &cobra.Command{
	Use:   "peer",
	Short: "display info about qri peers",
	Long:  ``,
}

var peerInfoCmd = &cobra.Command{
	Use:   "info",
	Short: `Get info on a qri peer`,
	Example: `  show info on a user named "mr0grog":
  $ qri peer info mr0grog`,
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

		res := &profile.Profile{}
		err = req.Info(p, res)
		ExitIfErr(err)

		data, err := json.MarshalIndent(res, "", "  ")
		ExitIfErr(err)

		printSuccess(string(data))
	},
}

func init() {
	// peerInfoCmd.Flags().StringP("format", "f", "", "set output format [json]")
	peerCmd.AddCommand(peerInfoCmd)

	RootCmd.AddCommand(peerCmd)
}
