package cmd

import (
	"fmt"

	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	searchCmdReindex bool
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for datasets",
	Long:  `Search looks through all of your namespaces for terms that match your query`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 && !searchCmdReindex {
			ErrExit(fmt.Errorf("wrong number of arguments. expected qri search [query]"))
		}

		req := core.NewSearchRequests(GetRepo(false))

		if searchCmdReindex {
			PrintInfo("building index...")
			done := false
			err := req.Reindex(&core.ReindexSearchParams{}, &done)
			ExitIfErr(err)
			PrintSuccess("reindex complete")
			if len(args) == 0 {
				return
			}
		}

		p := &repo.SearchParams{
			Q:      args[0],
			Limit:  30,
			Offset: 0,
		}
		res := []*repo.DatasetRef{}

		err := req.Search(p, &res)
		ExitIfErr(err)

		for i, ref := range res {
			PrintDatasetRefInfo(i, ref)
		}
	},
}

func init() {
	searchCmd.Flags().BoolVarP(&searchCmdReindex, "reindex", "r", false, "re-generate search index from scratch. might take a while.")
	RootCmd.AddCommand(searchCmd)
}
