package cmd

import (
	"fmt"
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
			fmt.Println("please specify a dataset to get the info of")
			return
		}
		PrintNotYetFinished(cmd)

		// ds, err := ipfs.NewDatastore()
		// ExitIfErr(err)

		// path := datastore.NewKey(args[0])
		// ns := LoadNamespaceGraph()

		// for n, resource := range ns {
		// 	if args[0] == n {
		// 		path = resource
		// 		break
		// 	}
		// }

		// rqg := LoadResourceQueriesGraph()
		// for p, res := range rqg {
		// 	if p.Equal(path) {
		// 		for i, q := range res {
		// 			iface, err := ds.Get(q)
		// 			ExitIfErr(err)
		// 			q, err := dataset.UnmarshalQuery(iface)
		// 			ExitIfErr(err)
		// 			s := q.Statement
		// 			spaces := ""
		// 			if len(s) > 40 {
		// 				s = s[:40]
		// 			} else {
		// 				spaces = strings.Repeat(" ", 40-len(s))
		// 			}
		// 			fmt.Printf("%d. %s%s%s\n", i+1, s, spaces, path.String())
		// 		}
		// 	}
		// }
	},
}

func init() {
	RootCmd.AddCommand(queriesCmd)
}
