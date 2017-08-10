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

	"github.com/ipfs/go-datastore"
	ipfs "github.com/qri-io/castore/ipfs"
	"github.com/qri-io/dataset"
	// "github.com/qri-io/dataset/datatypes"
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
			resource *dataset.Resource
			results  []byte
		)
		rgraph := LoadQueryResultsGraph()
		rqgraph := LoadResourceQueriesGraph()
		ns := LoadNamespaceGraph()

		ds, err := ipfs.NewDatastore()
		ExitIfErr(err)

		// TODO - make format output the parsed statement as well
		// to avoid triple-parsing
		sqlstr, _, remap, err := sql.Format(args[0])
		ExitIfErr(err)

		q := &dataset.Query{
			Syntax:    "sql",
			Resources: map[string]datastore.Key{},
			Statement: sqlstr,
			// TODO - set query schema
		}

		// collect table references
		for mapped, ref := range remap {
			// for i, adr := range stmt.References() {
			if ns[ref].String() == "" {
				ErrExit(fmt.Errorf("couldn't find resource for table name: %s", ref))
			}
			q.Resources[mapped] = ns[ref]
		}

		qData, err := q.MarshalJSON()
		ExitIfErr(err)

		qhash, err := ds.AddAndPinBytes(qData)
		ExitIfErr(err)
		fmt.Printf("query hash: %s\n", qhash)
		qpath := datastore.NewKey("/ipfs/" + qhash)

		cache := rgraph[qpath]

		if len(cache) > 0 {
			fmt.Println("returning hashed result.")
			resource, err = GetResource(ds, cache[0])
			if err != nil {
				results, err = GetStructuredData(ds, resource.Path)
			}
		}

		format, err := dataset.ParseDataFormatString(cmd.Flag("format").Value.String())
		if err != nil {
			ErrExit(fmt.Errorf("invalid data format: %s", cmd.Flag("format").Value.String()))
		}
		resource, results, err = sql.ExecQuery(ds, q, func(o *sql.ExecOpt) {
			o.Format = format
		})
		ExitIfErr(err)

		resource.Query = qpath

		resultshash, err := ds.AddAndPinBytes(results)
		ExitIfErr(err)
		fmt.Printf("results hash: %s\n", resultshash)

		resource.Path = datastore.NewKey("/ipfs/" + resultshash)

		rbytes, err := resource.MarshalJSON()
		ExitIfErr(err)

		rhash, err := ds.AddAndPinBytes(rbytes)
		fmt.Printf("result resource hash: %s\n", rhash)

		rgraph.AddResult(qpath, datastore.NewKey("/ipfs/"+rhash))
		err = SaveQueryResultsGraph(rgraph)
		ExitIfErr(err)

		for _, key := range q.Resources {
			rqgraph.AddQuery(key, qpath)
		}
		err = SaveResourceQueriesGraph(rqgraph)
		ExitIfErr(err)

		PrintResults(resource, results, format)
	},
}

func init() {
	RootCmd.AddCommand(runCmd)
	// runCmd.Flags().StringP("save", "s", "", "save the resulting dataset to a given address")
	runCmd.Flags().StringP("format", "f", "csv", "set output format [csv,json]")
}
