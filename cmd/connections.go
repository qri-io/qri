package cmd

import (
	"fmt"

	"github.com/qri-io/qri/repo/profile"
	"github.com/spf13/cobra"
)

// connectionsCmd lists
var connectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: `list open connections with qri & IPFS peers`,
	Example: `  show open qri connections:
  $ qri connections

  show all IPFS connections:
  $ qri connections --ipfs`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			ErrExit(fmt.Errorf("connections accepts no arguments"))
		}
		req, err := peerRequests(true)
		ExitIfErr(err)

		if cmd.Flag("ipfs").Value.String() == "true" {
			limit := 200
			res := []string{}
			err := req.ConnectedIPFSPeers(&limit, &res)
			ExitIfErr(err)
			for i, p := range res {
				printSuccess("%d.\t%s", i+1, p)
			}
		} else {
			limit := 200
			res := []*profile.Profile{}
			err := req.ConnectedQriProfiles(&limit, &res)
			ExitIfErr(err)

			i := 0
			for _, p := range res {
				printSuccess("%d.\t%s\t%s", i+1, p.ID, p.Peername)
				i++
			}
		}
	},
}

func init() {
	// connectionsCmd.Flags().StringP("format", "f", "", "set output format [json]")
	connectionsCmd.Flags().BoolP("ipfs", "", false, "show ipfs peers")

	RootCmd.AddCommand(connectionsCmd)
}
