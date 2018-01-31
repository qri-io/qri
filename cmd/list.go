package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	dsListLimit, dsListOffset int
)

var datasetListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "show a list of datasets",
	Long: `
list shows lists of datasets, including names and current hashes. 

The default list is the latest version of all datasets you have on your local 
qri repository.`,
	Example: `  show all of your datasets:
  $ qri list`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			// TODO - add limit & offset params
			r, err := datasetRequests(false)
			ExitIfErr(err)

			p := &core.ListParams{
				Limit:  dsListLimit,
				Offset: dsListOffset,
			}
			refs := []*repo.DatasetRef{}
			err = r.List(p, &refs)
			ExitIfErr(err)

			outformat := cmd.Flag("format").Value.String()
			switch outformat {
			case "":
				for _, ref := range refs {
					printInfo("%s\t\t\t: %s", ref.Name, ref.Path)
				}
			case dataset.JSONDataFormat.String():
				data, err := json.MarshalIndent(refs, "", "  ")
				ExitIfErr(err)
				fmt.Printf("%s\n", string(data))
			default:
				ErrExit(fmt.Errorf("unrecognized format: %s", outformat))
			}
		} else {
			// for _, ref := range args {
			// ref, err := repo.ParseDatasetRef(ref)
			// ExitIfErr(err)
			// }
			r, err := datasetRequests(true)
			ExitIfErr(err)

			p := &core.ListParams{
				Peername: args[0],
				Limit:    dsListLimit,
				Offset:   dsListOffset,
			}
			refs := []*repo.DatasetRef{}
			err = r.List(p, &refs)
			ExitIfErr(err)

			outformat := cmd.Flag("format").Value.String()
			switch outformat {
			case "":
				if len(refs) == 0 {
					printInfo("%s has no datasets", args[0])
				}
				for _, ref := range refs {
					printInfo("%s\t\t: %s", ref.Name, ref.Path)
				}
			case dataset.JSONDataFormat.String():
				data, err := json.MarshalIndent(refs, "", "  ")
				ExitIfErr(err)
				fmt.Printf("%s\n", string(data))
			default:
				ErrExit(fmt.Errorf("unrecognized format: %s", outformat))
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(datasetListCmd)
	datasetListCmd.Flags().StringP("format", "f", "", "set output format [json]")
	datasetListCmd.Flags().IntVarP(&dsListLimit, "limit", "l", 25, "limit results, default 25")
	datasetListCmd.Flags().IntVarP(&dsListOffset, "offset", "o", 0, "offset results, default 0")
}
