package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/qri-io/qri/repo/profile"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"get", "describe"},
	Short:   "Show summarized description of a dataset",
	Long:    `info describes users and datasets`,
	Example: `  show b5 user info
	get info for b5/comics:
  $ qri info b5/comics

  get info for a dataset at a specific version:
  $ qri info QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// ErrExit(fmt.Errorf("please specify a dataset path or name to get the info of"))
			profileGetCmd.Run(&cobra.Command{}, []string{})
			return
		}

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

		online := false
		// check to see if we're all local
		r := getRepo(false)
		for _, arg := range args {
			ref, err := repo.ParseDatasetRef(arg)
			ExitIfErr(err)
			local, err := repo.IsLocalRef(r, ref)
			ExitIfErr(err)
			if !local {
				online = true
			}
		}

		pr, err := peerRequests(online)
		ExitIfErr(err)

		req, err := datasetRequests(online)
		ExitIfErr(err)

		for i, arg := range args {
			ref, err := repo.ParseDatasetRef(arg)
			ExitIfErr(err)

			if ref.IsPeerRef() {
				p := &core.PeerInfoParams{
					Peername: ref.Peername,
				}
				res := &profile.Profile{}
				err := pr.Info(p, res)
				ExitIfErr(err)

				if outformat == "" {
					printPeerInfo(0, res)
				} else {
					data, err := json.MarshalIndent(res, "", "  ")
					ExitIfErr(err)
					fmt.Printf("%s", string(data))
				}
			} else {
				res := &repo.DatasetRef{}
				err = req.Get(ref, res)
				ExitIfErr(err)

				if outformat == "" {
					printDatasetRefInfo(i, res)
				} else {
					data, err := json.MarshalIndent(res.Dataset, "", "  ")
					ExitIfErr(err)
					fmt.Printf("%s", string(data))
				}
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(infoCmd)
	infoCmd.Flags().StringP("format", "f", "", "set output format [json]")
}
