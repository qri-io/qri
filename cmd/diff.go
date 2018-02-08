package cmd

import (
	"fmt"

	// "github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/datasetDiffer"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
	// diff "github.com/yudai/gojsondiff"
)

var datasetDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "diff two datasets",
	Long: `
Diff compares two datasets from your repo and prints a represntation 
of the differences between them.  You can specifify the datasets
either by name or by their hash`,
	Example: `todo`,
	Run: func(cmd *cobra.Command, args []string) {
		for i, arg := range args {
			fmt.Printf("%d: %s\n", i, arg)
		}
		if len(args) < 2 {
			ErrExit(fmt.Errorf("please provide names for two datsets"))
		}

		leftRef, err := repo.ParseDatasetRef(args[0])
		ExitIfErr(err)

		rightRef, err := repo.ParseDatasetRef(args[1])
		ExitIfErr(err)

		req, err := datasetRequests(false)
		ExitIfErr(err)

		left := repo.DatasetRef{}
		right := repo.DatasetRef{}

		err = req.Get(&leftRef, &left)
		ExitIfErr(err)

		err = req.Get(&rightRef, &right)
		ExitIfErr(err)

		diffs := make(map[string]*datasetDiffer.SubDiff)

		p := &core.DiffParams{
			DsLeft:  left.Dataset,
			DsRight: right.Dataset,
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
		result, err := datasetDiffer.MapDiffsToString(diffs, displayFormat)
		ExitIfErr(err)
		fmt.Println(result)
	},
}

func init() {
	RootCmd.AddCommand(datasetDiffCmd)
	datasetDiffCmd.Flags().StringP("display", "d", "", "set display format [reg|short|delta|detail]")
	// datasetDiffCmd.Flags().BoolP("color", "c", false, "set ")
}
