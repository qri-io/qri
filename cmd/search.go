// Copyright © 2016 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"github.com/qri-io/namespace"
	"github.com/spf13/cobra"
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for datasets",
	Long:  `Search looks through all of your namespaces for terms that match your query`,
	Run: func(cmd *cobra.Command, args []string) {
		var q string
		if len(args) > 0 {
			q = args[0]
		}

		namespaces := GetNamespaces(cmd, args)
		// tasks := len(namespaces)
		// done := make(chan int)
		for _, ns := range namespaces {
			// go func(done chan int) {
			if s, ok := ns.Namespace.(namespace.SearchableNamespace); ok {
				results, err := namespace.ReadAllDatasets(s.Search(q, -1, 0))
				if err != nil {
					PrintErr(err)
				} else {
					PrintNamespace(ns)
					if len(results) > 0 {
						for i, ds := range results {
							PrintDatasetShortInfo(i+1, ds)
						}
					} else {
						PrintInfo("no results.")
					}
				}
				fmt.Println()
			} else {
				PrintWarning("namspace %s doesn't support searching", ns.String())
			}
			// 	done <- i
			// }(done)
		}
		// for {
		// 	<-done
		// 	tasks -= 1
		// 	if tasks == 0 {
		// 		return
		// 	}
		// }
	},
}

func init() {
	RootCmd.AddCommand(searchCmd)
}
