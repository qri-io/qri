package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"get", "describe"},
	Short:   "Show summarized description of a dataset",
	Long: `
Usage:
	qri info <dataset ref…>

Feel free to add multiple dataset names to show more than one summary`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			ErrExit(fmt.Errorf("please specify a dataset path or name to get the info of"))
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

		req, err := datasetRequests(false)
		ExitIfErr(err)

		for i, arg := range args {
			rt, ref := dsfs.RefType(arg)
			p := &core.GetDatasetParams{}
			switch rt {
			case "path":
				p.Path = datastore.NewKey(ref)
			case "name":
				p.Name = ref
			}
			res := &repo.DatasetRef{}
			err := req.Get(p, res)
			ExitIfErr(err)
			if outformat == "" {
				printDatasetRefInfo(i, res)
			} else {
				data, err := json.MarshalIndent(res.Dataset, "", "  ")
				ExitIfErr(err)
				fmt.Printf("%s", string(data))
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(infoCmd)
	infoCmd.Flags().StringP("format", "f", "", "set output format [json]")
}
