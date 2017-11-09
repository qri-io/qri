package cmd

import (
	"fmt"

	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/fs"
	"github.com/spf13/cobra"
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for datasets",
	Long:  `Search looks through all of your namespaces for terms that match your query`,
	Run: func(cmd *cobra.Command, args []string) {
		reindex, err := cmd.Flags().GetBool("reindex")
		if err != nil {
			fmt.Printf("error: %s", err.Error())
		}
		ExitIfErr(err)

		if len(args) != 1 && !reindex {
			ErrExit(fmt.Errorf("wrong number of arguments. expected qri search [query]"))
		}

		r := GetRepo(false)
		store := GetIpfsFilestore(false)
		req := core.NewSearchRequests(store, r)

		if reindex {
			if fsr, ok := r.(*fs_repo.Repo); ok {
				PrintInfo("building index...")
				err = fsr.UpdateSearchIndex(store)
				if err != nil {
					fmt.Printf("error: %s", err.Error())
				}
				ExitIfErr(err)
			}
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
		err = req.Search(p, &res)
		ExitIfErr(err)

		for i, ref := range res {
			PrintDatasetRefInfo(i, ref)
		}
	},
}

func init() {
	searchCmd.Flags().BoolP("reindex", "r", false, "re-generate search index from scratch. might take a while.")
	RootCmd.AddCommand(searchCmd)
}
