package cmd

import (
	"fmt"

	"github.com/qri-io/dsdiff"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var datasetDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "compare differences between two datasets",
	Long: `
Diff compares two datasets from your repo and prints a represntation 
of the differences between them.  You can specifify the datasets
either by name or by their hash`,
	// Example: `todo`,
	Annotations: map[string]string{
		"group": "dataset",
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		for i, arg := range args {
			fmt.Printf("%d: %s\n", i, arg)
		}
		requireNotRPC(cmd.Name())

		req, err := datasetRequests(false)
		ExitIfErr(err)

		left, err := repo.ParseDatasetRef(args[0])
		ExitIfErr(err)
		right, err := repo.ParseDatasetRef(args[1])
		ExitIfErr(err)

		diffs := make(map[string]*dsdiff.SubDiff)

		p := &core.DiffParams{
			Left:    left,
			Right:   right,
			DiffAll: true,
		}

		err = req.Diff(p, &diffs)
		ExitIfErr(err)
		displayFormat := "listKeys"
		displayFlag := cmd.Flag("display").Value.String()
		if displayFlag != "" {
			switch displayFlag {
			case "reg", "regular":
				displayFormat = "listKeys"
			case "short", "s":
				displayFormat = "simple"
			case "delta":
				displayFormat = "delta"
			case "detail":
				displayFormat = "plusMinus"
			}
		}

		result, err := dsdiff.MapDiffsToString(diffs, displayFormat)
		ExitIfErr(err)

		printDiffs(result)
	},
}

func init() {
	RootCmd.AddCommand(datasetDiffCmd)
	datasetDiffCmd.Flags().StringP("display", "d", "", "set display format [reg|short|delta|detail]")
	// datasetDiffCmd.Flags().BoolP("color", "c", false, "set ")
}
