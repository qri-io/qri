package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/qri-io/dataset"
	"github.com/spf13/cobra"
)

var datasetListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "list your local datasets",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO - add limit & offset params
		refs, err := GetRepo(false).Namespace(100, 0)
		ExitIfErr(err)

		outformat := cmd.Flag("format").Value.String()

		switch outformat {
		case "":
			for _, ref := range refs {
				PrintInfo("%s\t\t\t: %s", ref.Name, ref.Path)
			}
		case dataset.JSONDataFormat.String():
			data, err := json.MarshalIndent(refs, "", "  ")
			ExitIfErr(err)
			fmt.Printf("%s\n", string(data))
		default:
			ErrExit(fmt.Errorf("unrecognized format: %s", outformat))
		}

	},
}

func init() {
	RootCmd.AddCommand(datasetListCmd)
	datasetListCmd.Flags().StringP("format", "f", "", "set output format [json]")
}
