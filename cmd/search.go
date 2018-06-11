package cmd

import (
	"fmt"

	"github.com/qri-io/qri/core"
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
		p := &core.SearchParams{
			QueryString: args[0],
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
	RootCmd.AddCommand(searchCmd)
}

// import (
//  "encoding/json"
//  "fmt"

//  "github.com/qri-io/dataset"
//  "github.com/qri-io/qri/core"
//  "github.com/qri-io/qri/repo"
//  "github.com/spf13/cobra"
// )

// var (
//  searchCmdReindex bool
// )

// // searchCmd represents the search command
// var searchCmd = &cobra.Command{
//  Use:   "search",
//  Short: "Search qri",
//  Long:  `Search datasets & peers that match your query`,
//  Annotations: map[string]string{
//    "group": "other",
//  },
//  PreRun: func(cmd *cobra.Command, args []string) {
//    loadConfig()
//  },
//  Run: func(cmd *cobra.Command, args []string) {
//    if len(args) != 1 && !searchCmdReindex {
//      ErrExit(fmt.Errorf("wrong number of arguments. expected qri search [query]"))
//    }

//    req, err := searchRequests(false)
//    ExitIfErr(err)

//    if searchCmdReindex {
//      printInfo("building index...")
//      done := false
//      err := req.Reindex(&core.ReindexSearchParams{}, &done)
//      ExitIfErr(err)
//      printSuccess("reindex complete")
//      if len(args) == 0 {
//        return
//      }
//    }

//    p := &repo.SearchParams{
//      Q:      args[0],
//      Limit:  30,
//      Offset: 0,
//    }
//    res := []repo.DatasetRef{}

//    err = req.Search(p, &res)
//    ExitIfErr(err)

//    outformat := cmd.Flag("format").Value.String()

//    switch outformat {
//    case "":
//      for i, ref := range res {
//        printDatasetRefInfo(i, ref)
//      }
//    case dataset.JSONDataFormat.String():
//      data, err := json.MarshalIndent(res, "", "  ")
//      ExitIfErr(err)
//      fmt.Printf("%s\n", string(data))
//    default:
//      ErrExit(fmt.Errorf("unrecognized format: %s", outformat))
//    }

//  },
// }

// func init() {
//  searchCmd.Flags().BoolVarP(&searchCmdReindex, "reindex", "r", false, "re-generate search index from scratch. might take a while.")
//  searchCmd.Flags().StringP("format", "f", "", "set output format [json]")
//  // RootCmd.AddCommand(searchCmd)
// }
