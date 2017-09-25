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
	"github.com/qri-io/qri/core/search"
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

		fs, err := GetIpfsFilestore()
		ExitIfErr(err)

		results, err := search.Search(GetRepo(), fs, search.NewDatasetQuery(args[0], 30, 0))
		ExitIfErr(err)

		if len(results) > 0 {
			for i, ds := range results {
				PrintDatasetRefInfo(i+1, ds)
			}
		} else {
			PrintWarning("no results")
		}
	},
}

func init() {
	RootCmd.AddCommand(searchCmd)
}

// q := strings.ToLower(args[0])
// limit := 30
// results := make([]*repo.DatasetRef, limit)
// r := GetRepo()

// store, err := GetIpfsFilestore()
// ExitIfErr(err)

// ns, err := r.Namespace(1000, 0)
// ExitIfErr(err)
// i := 0

// for name, path := range ns {
// 	if i == limit {
// 		break
// 	}

// 	ds, err := dsfs.LoadDataset(store, path)
// 	if err != nil {
// 		PrintWarning("skipped dataset: %s", name, err.Error())
// 	}

// 	if strings.Contains(strings.ToLower(name), q) ||
// 		strings.Contains(strings.ToLower(ds.Title), q) ||
// 		strings.Contains(strings.ToLower(ds.Description), q) {

// 		ref := &repo.DatasetRef{
// 			Name:    name,
// 			Path:    path,
// 			Dataset: ds,
// 		}
// 		results[i] = ref
// 		i++
// 	}
// }
