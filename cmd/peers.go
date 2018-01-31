package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/qri-io/qri/repo/profile"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
)

// peersCmd represents the info command
var peersCmd = &cobra.Command{
	Use:   "peers",
	Short: "List known qri peers",
	Long:  `peers lists the peers your qri node has seen before`,
	Example: `  list qri peers:
  $ qri peers`,
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

		res := []*profile.Profile{}
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

func init() {
	RootCmd.AddCommand(peersCmd)
	peersCmd.Flags().StringP("format", "f", "", "set output format [json]")
}
