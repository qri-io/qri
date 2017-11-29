package cmd

import (
	"fmt"

	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var datasetRenameCmd = &cobra.Command{
	Use:     "rename",
	Aliases: []string{"mv"},
	Short:   "rename a dataset from your local namespace based on a resource hash",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			ErrExit(fmt.Errorf("please provide current & new dataset names"))
		}

		req := core.NewDatasetRequests(GetRepo(false))
		p := &core.RenameParams{
			Current: args[0],
			New:     args[1],
		}
		res := &repo.DatasetRef{}
		err := req.Rename(p, res)
		ExitIfErr(err)

		PrintSuccess("renamed dataset %s", res.Name)
	},
}

func init() {
	RootCmd.AddCommand(datasetRenameCmd)
}
