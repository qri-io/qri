package cmd

import (
	"fmt"

	"github.com/qri-io/qri/core"
	"github.com/qri-io/registry/regclient"
	"github.com/spf13/cobra"
)

var (
	searchCmdReindex bool
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search qri",
	Long:  `Search datasets & peers that match your query`,
	Annotations: map[string]string{
		"group": "other",
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 && !searchCmdReindex {
			ErrExit(fmt.Errorf("wrong number of arguments. expected qri search [query]"))
		}

		req, err := searchRequests(true)
		ExitIfErr(err)
		p := &regclient.SearchParams{
			QueryString: args[0],
			Filters:     nil,
			Limit:       100,
			Offset:      0}

		results := &[]core.SearchResult{}
		err = req.Search(p, results)
		ExitIfErr(err)
		fmt.Printf("showing %d results for '%s'\n", len(*results), args[0])
		for i, result := range *results {
			printSearchResult(i, result)
		}

	},
}

func init() {
	// searchCmd.Flags().BoolVarP(&searchCmdReindex, "reindex", "r", false, "re-generate search index from scratch. might take a while.")
	// searchCmd.Flags().StringP("format", "f", "", "set output format [json]")
	RootCmd.AddCommand(searchCmd)
}
