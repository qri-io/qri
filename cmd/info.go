package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"get", "describe"},
	Short:   "show summarized description of a dataset",
	Long:    `info describes datasets`,
	Example: `  get info for b5/comics:
  $ qri info b5/comics

  get info for a dataset at a specific version:
  $ qri info me@/ipfs/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn

  or

  $ qri info me/comics@/ipfs/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn`,
	Annotations: map[string]string{
		"group": "dataset",
	},
	Args: cobra.MinimumNArgs(1),
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

		req, err := datasetRequests(false)
		ExitIfErr(err)

		for i, arg := range args {
			ref, err := repo.ParseDatasetRef(arg)
			ExitIfErr(err)

			if ref.IsPeerRef() {
				printWarning("please specify a dataset for peer %s", ref.Peername)
			} else {
				res := repo.DatasetRef{}
				err = req.Get(&ref, &res)
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
