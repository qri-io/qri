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
	// "io/ioutil"

	// "encoding/json"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/datatypes"
	sql "github.com/qri-io/dataset_sql"
	"github.com/qri-io/ipfs_datastore"
	"github.com/spf13/cobra"
	// "gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore"
	// "gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore/query"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a query",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		// if len(args) == 0 {
		// 	ErrExit(fmt.Errorf("Please provide a query or address to execute"))
		// }
		ds, err := ipfs_datastore.NewDatastore()
		ExitIfErr(err)

		hhash, err := ds.AddAndPinPath("testdata/hours.csv")
		ExitIfErr(err)

		fmt.Printf("hours hash: %s\n", hhash)

		r := &dataset.Resource{
			Format: dataset.CsvDataFormat,
			Schema: &dataset.Schema{
				Fields: []*dataset.Field{
					&dataset.Field{Name: "field_1", Type: datatypes.Date},
					&dataset.Field{Name: "field_3", Type: datatypes.Float},
					&dataset.Field{Name: "field_3", Type: datatypes.String},
					&dataset.Field{Name: "field_4", Type: datatypes.String},
				},
			},
			Path: datastore.NewKey("/ipfs/" + hhash),
		}

		rdata, err := r.MarshalJSON()
		ExitIfErr(err)

		hash, err := ds.AddAndPinBytes(rdata)
		ExitIfErr(err)

		// fmt.Printf("resource hash: %s\n", hash)
		// store := localRepo.Datastore()
		// res, err := store.Query(query.Query{
		// 	Prefix:   "",
		// 	KeysOnly: true,
		// 	// Limit:    500,
		// })
		// entries, err := res.Rest()
		// ExitIfErr(err)
		// for _, e := range entries {
		// 	fmt.Println(e.Key)
		// }

		key := datastore.NewKey("/ipfs/" + hash)
		fmt.Printf("resource hash: %s\n", key.String())

		// idata, err := ds.Get(key)
		// ExitIfErr(err)

		// data, ok := idata.([]byte)
		// if !ok {
		// 	ErrExit(fmt.Errorf("data is not a byte slice"))
		// }

		// ds.AddAndPinFile("", data)

		// data, err := store.Get(datastore.NewKey(hash))
		// data, err := ioutil.ReadAll(rdr)
		// ExitIfErr(err)

		// r2 := &dataset.Resource{}
		// err = r2.UnmarshalJSON(data)
		// ExitIfErr(err)

		// fmt.Println(r2)

		q := &dataset.Query{
			Syntax: "sql",
			Resources: map[string]datastore.Key{
				"a": key,
			},
			Statement: "select field_1 from a",
		}

		qData, err := q.MarshalJSON()
		ExitIfErr(err)

		qhash, err := ds.AddAndPinBytes(qData)
		ExitIfErr(err)
		fmt.Printf("query hash: %s\n", qhash)

		// q.UnmarshalJSON([]byte(args[0]))
		// ExitIfErr(err)

		// format, err := dataset.ParseDataFormatString(cmd.Flag("format").Value.String())
		// if err != nil {
		// 	ErrExit(fmt.Errorf("invalid data format: %s", cmd.Flag("format").Value.String()))
		// }
		resource, results, err := sql.ExecQuery(ds, q, func(o *sql.ExecOpt) {
			// o.Format = format
			o.Format = dataset.CsvDataFormat
		})
		fmt.Println(resource)
		fmt.Println(string(results))
		fmt.Println(err)
		ExitIfErr(err)

		rbytes, err := resource.MarshalJSON()
		ExitIfErr(err)

		rhash, err := ds.AddAndPinBytes(rbytes)
		fmt.Printf("result hash: %s\n", rhash)

		resultshash, err := ds.AddAndPinBytes(results)
		ExitIfErr(err)

		fmt.Printf("results hash: %s\n", resultshash)

		// stmt, err := query.Parse(args[0])
		// ExitIfErr(err)

		// adr := dataset.NewAddress("")
		// if save := cmd.Flag("save").Value.String(); save != "" {
		// 	if !dataset.ValidAddressString(save) {
		// 		PrintErr(fmt.Errorf("'%s' is not a valid address string to save to", save))
		// 		os.Exit(-1)
		// 	}
		// 	adr = dataset.NewAddress(save)
		// }

		// results, data, err := stmt.Exec(GetNamespaces(cmd, args), func(o *query.ExecOpt) {
		// results, data, err := stmt.Exec(LocalNamespaces(cmd, args), func(o *query.ExecOpt) {
		// 	o.Format = format
		// })
		// ExitIfErr(err)

		// if !adr.IsEmpty() {
		// 	results.Address = adr
		// 	err := WriteDataset(Cache(), results, map[string][]byte{
		// 		fmt.Sprintf("%s.%s", adr.String(), format.String()): data,
		// 	})

		// 	ExitIfErr(err)
		// 	PrintSuccess("results saved to: %s%s", cachePath(), DatasetPath(results))
		// 	return
		// }

		// PrintResults(resource, data, format)
	},
}

func init() {
	RootCmd.AddCommand(runCmd)
	runCmd.Flags().StringP("save", "s", "", "save the resulting dataset to a given address")
	runCmd.Flags().StringP("format", "f", "csv", "set output format [csv,json]")
}
