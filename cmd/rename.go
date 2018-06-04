package cmd

import (
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var datasetRenameCmd = &cobra.Command{
	Use:     "rename",
	Aliases: []string{"mv"},
	Short:   "change the name of a dataset",
	Long: `
Rename changes the name of a dataset.

Note that if someone has added your dataset to their qri node, and then
you rename your local dataset, your peer's version of your dataset will
not have the updated name. While this won't break anything, it will
confuse anyone who has added your dataset before the change. Try to keep
renames to a minimum.`,
	Example: `  rename a dataset named annual_pop to annual_population:
  $ qri rename me/annual_pop me/annual_population`,
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Annotations: map[string]string{
		"group": "dataset",
	},
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		current, err := repo.ParseDatasetRef(args[0])
		ExitIfErr(err)

		next, err := repo.ParseDatasetRef(args[1])
		ExitIfErr(err)

		req, err := datasetRequests(false)
		ExitIfErr(err)
		p := &core.RenameParams{
			Current: current,
			New:     next,
		}
		res := repo.DatasetRef{}
		err = req.Rename(p, &res)
		ExitIfErr(err)

		printSuccess("renamed dataset %s", res.Name)
	},
}

func init() {
	RootCmd.AddCommand(datasetRenameCmd)
}
