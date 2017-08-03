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

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/datatypes"
	// sql "github.com/qri-io/dataset_sql"
	"github.com/spf13/cobra"
	"gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore"
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
		}

		rdata, err := r.MarshalJSON()
		ExitIfErr(err)

		hash, err := addAndPinFile("resource.json", rdata)
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

		data, err := getKey(datastore.NewKey("/ipfs/" + hash))
		ExitIfErr(err)

		// data, err := store.Get(datastore.NewKey(hash))
		// data, err := ioutil.ReadAll(rdr)
		// ExitIfErr(err)

		fmt.Println(string(data))

		// r2 := &dataset.Resource{}
		// err = r2.UnmarshalJSON(data)
		// ExitIfErr(err)

		// fmt.Println(r2)

		// q := &dataset.Query{
		// 	Syntax: "sql",
		// 	Resources: map[string]datastore.Key{
		// 		"a": datastore.NewKey(""),
		// 	},
		// 	Statement: "select field_1 from a",
		// }

		// q.UnmarshalJSON([]byte(args[0]))
		// ExitIfErr(err)

		// format, err := dataset.ParseDataFormatString(cmd.Flag("format").Value.String())
		// if err != nil {
		// 	ErrExit(fmt.Errorf("invalid data format: %s", cmd.Flag("format").Value.String()))
		// }

		// resource, data, err := sql.ExecQuery(repo.Datastore(), q, func(o *sql.ExecOpt) {
		// 	o.Format = format
		// })
		// ExitIfErr(err)

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
