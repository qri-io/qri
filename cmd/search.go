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
		if len(args) != 1 {
			ErrExit(fmt.Errorf("wrong number of arguments. expected qri search [query]"))
		}

		PrintWarning("CLI search only supports searching local datasets for now")

		fs, err := GetIpfsFilestore(false)
		ExitIfErr(err)

		r := GetRepo(false)

		reindex, err := cmd.Flags().GetBool("reindex")
		ExitIfErr(err)

		if reindex {
			if fsr, ok := r.(*fs_repo.Repo); ok {
				PrintInfo("building index...")
				err = fsr.UpdateSearchIndex(fs)
				ExitIfErr(err)
			}
		}

		PrintInfo("running search...")
		if s, ok := r.(repo.Searchable); ok {
			res, err := s.Search(args[0])
			ExitIfErr(err)
			PrintInfo(res)
		} else {
			ErrExit(fmt.Errorf("this repository doesn't support search"))
		}

		// results, err := search.Search(GetRepo(), fs, search.NewDatasetQuery(args[0], 30, 0))
		// ExitIfErr(err)

		// if len(results) > 0 {
		// 	for i, ds := range results {
		// 		PrintDatasetRefInfo(i+1, ds)
		// 	}
		// } else {
		// 	PrintWarning("no results")
		// }
	},
}

func init() {
	searchCmd.Flags().BoolP("reindex", "r", false, "re-generate search index from scratch. might take a while.")
	RootCmd.AddCommand(searchCmd)
}
