package cmd

import (
	"fmt"

	// "github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/datasetDiffer"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
	diff "github.com/yudai/gojsondiff"
)

var datasetDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "diff two datasets",
	Long: `
Diff diffs two datasets`,
	Example: `todo`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			ErrExit(fmt.Errorf("please provide names for two datsets"))
		}

		leftRef, err := repo.ParseDatasetRef(args[0])
		ExitIfErr(err)

		rightRef, err := repo.ParseDatasetRef(args[1])
		ExitIfErr(err)

		req, err := datasetRequests(false)
		ExitIfErr(err)

		left := &repo.DatasetRef{}
		right := &repo.DatasetRef{}

		err = req.Get(leftRef, left)
		ExitIfErr(err)

		err = req.Get(rightRef, right)
		ExitIfErr(err)

		diffs := &map[string]diff.Diff{}

		p := &core.DiffParams{
			DsLeft:  left.Dataset,
			DsRight: right.Dataset,
			DiffAll: true,
		}

		err = req.Diff(p, diffs)
		ExitIfErr(err)

		fmt.Println(datasetDiffer.MapDiffsToString(*diffs))

		// printSuccess("renamed dataset %s", res.Name)
	},
}

func init() {
	RootCmd.AddCommand(datasetDiffCmd)
}
