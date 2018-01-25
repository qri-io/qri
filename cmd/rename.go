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
	Short:   "show the history of changes to a dataset",
	Long: `
Rename changes the name of a dataset. So, uh, it’s worth noting that this can 
break lots of stuff for other people, especially in these early days of qri. 

So free to rename stuff lots at first, but try to settle on a name and stick 
with it, especially if you want other people to like your datasets.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			ErrExit(fmt.Errorf("please provide current & new dataset names"))
		}

		req, err := datasetRequests(false)
		ExitIfErr(err)
		p := &core.RenameParams{
			Current: args[0],
			New:     args[1],
		}
		res := &repo.DatasetRef{}
		err = req.Rename(p, res)
		ExitIfErr(err)

		printSuccess("renamed dataset %s", res.Name)
	},
}

func init() {
	RootCmd.AddCommand(datasetRenameCmd)
}
