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
	"time"

	"github.com/qri-io/dataset"
	sql "github.com/qri-io/dataset_sql"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a query",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			ErrExit(fmt.Errorf("Please provide a query string to execute"))
		}

		var (
			structure *dataset.Structure
			results   []byte
		)
		rgraph := LoadQueryResultsGraph()
		// rqgraph := LoadResourceQueriesGraph()
		ns := LoadNamespaceGraph()

		store, err := GetIpfsDatastore()
		ExitIfErr(err)

		// TODO - make format output the parsed statement as well
		// to avoid triple-parsing
		sqlstr, _, remap, err := sql.Format(args[0])
		ExitIfErr(err)

		ds := &dataset.Dataset{
			Timestamp:   time.Now().In(time.UTC),
			QuerySyntax: "sql",
			Resources:   map[string]*dataset.Dataset{},
			QueryString: sqlstr,
			// TODO - set query schema
		}

		// collect table references
		for mapped, ref := range remap {
			// for i, adr := range stmt.References() {
			if ns[ref].String() == "" {
				ErrExit(fmt.Errorf("couldn't find resource for table name: %s", ref))
			}
			d, err := dataset.LoadDataset(store, ns[ref])
			if err != nil {
				ErrExit(err)
			}
			ds.Resources[mapped] = d
		}

		// qData, err := q.MarshalJSON()
		// ExitIfErr(err)

		// qhash, err := store.AddAndPinBytes(qData)
		// ExitIfErr(err)
		// fmt.Printf("query hash: %s\n", qhash)
		// qpath := datastore.NewKey("/ipfs/" + qhash)

		// cache := rgraph[qpath]

		// if len(cache) > 0 {
		// 	fmt.Println("returning hashed result.")
		// 	resource, err = GetStructure(store, cache[0])
		// 	if err != nil {
		// 		results, err = GetStructuredData(store, resource.Path)
		// 	}
		// }

		format, err := dataset.ParseDataFormatString(cmd.Flag("format").Value.String())
		if err != nil {
			ErrExit(fmt.Errorf("invalid data format: %s", cmd.Flag("format").Value.String()))
		}
		structure, results, err = sql.Exec(store, ds, func(o *sql.ExecOpt) {
			o.Format = format
		})
		ExitIfErr(err)

		// TODO - move this into setting on the dataset outparam
		ds.Structure = structure
		ds.Length = len(results)

		ds.Data, err = store.Put(results)
		ExitIfErr(err)

		dspath, err := ds.Save(store)
		ExitIfErr(err)

		rgraph.AddResult(dspath, dspath)
		err = SaveQueryResultsGraph(rgraph)
		ExitIfErr(err)

		// TODO - restore
		// rqgraph, err := r.repo.ResourceQueries()
		// if err != nil {
		// 	return err
		// }

		// for _, key := range ds.Resources {
		// 	rqgraph.AddQuery(key, dspath)
		// }
		// err = r.repo.SaveResourceQueries(rqgraph)
		// if err != nil {
		// 	return err
		// }

		PrintResults(structure, results, format)
	},
}

func init() {
	RootCmd.AddCommand(runCmd)
	// runCmd.Flags().StringP("save", "s", "", "save the resulting dataset to a given address")
	runCmd.Flags().StringP("format", "f", "csv", "set output format [csv,json]")
}
