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
			req := core.NewQueryRequests(GetRepo(false))

			p := &core.ListParams{
				Limit:  30,
				Offset: 0,
			}

			res := []*repo.DatasetRef{}
			err := req.List(p, &res)
			ExitIfErr(err)

			for i, q := range res {
				PrintQuery(i, q)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(queriesCmd)
}
