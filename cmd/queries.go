package cmd

import (
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var queriesCmd = &cobra.Command{
	Use:     "queries",
	Aliases: []string{"qs"},
	Short:   "show queries related to a dataset",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			req, err := QueryRequests(false)
			ExitIfErr(err)
			p := core.NewListParams("-created", pageNum, pageSize)

			res := []*repo.DatasetRef{}
			err = req.List(&p, &res)
			ExitIfErr(err)

			for i, q := range res {
				PrintQuery(i, q)
			}
		}
	},
}

func init() {
	queriesCmd.Flags().IntVarP(&pageNum, "page", "p", 1, "page of results to show")
	queriesCmd.Flags().IntVarP(&pageSize, "size", "s", 30, "max number of results to show per page")
	RootCmd.AddCommand(queriesCmd)
}
