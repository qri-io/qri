package cmd

import (
	// "encoding/json"
	// "fmt"
	// "github.com/qri-io/dataset"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	dsLogLimit, dsLogOffset int
	dsLogName               string
)

var datasetLogCmd = &cobra.Command{
	Use: "log",
	// Aliases: []string{"ls"},
	Short: "show log of dataset history",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO - add limit & offset params
		r, err := historyRequests(false)
		ExitIfErr(err)

		p := &core.LogParams{
			// Limit:  dsLogLimit,
			// Offset: dsLogOffset,
			Name: dsLogName,
		}
		refs := []*repo.DatasetRef{}
		err = r.Log(p, &refs)
		ExitIfErr(err)

		for _, ref := range refs {
			printSuccess("%s - %s\n\t%s\n", ref.Dataset.Commit.Timestamp.Format("Jan _2 15:04:05"), ref.Path, ref.Dataset.Commit.Title)
		}

		// outformat := cmd.Flag("format").Value.String()
		// switch outformat {
		// case "":
		// 	for _, ref := range refs {
		// 		printInfo("%s\t\t\t: %s", ref.Name, ref.Path)
		// 	}
		// case dataset.JSONDataFormat.String():
		// 	data, err := json.MarshalIndent(refs, "", "  ")
		// 	ExitIfErr(err)
		// 	fmt.Printf("%s\n", string(data))
		// default:
		// 	ErrExit(fmt.Errorf("unrecognized format: %s", outformat))
		// }

	},
}

func init() {
	RootCmd.AddCommand(datasetLogCmd)
	// datasetLogCmd.Flags().StringP("format", "f", "", "set output format [json]")
	datasetLogCmd.Flags().IntVarP(&dsLogLimit, "limit", "l", 25, "limit results, default 25")
	datasetLogCmd.Flags().IntVarP(&dsLogOffset, "offset", "o", 0, "offset results, default 0")
	datasetLogCmd.Flags().StringVarP(&dsLogName, "name", "n", "", "name of dataset to get logs for")
}
