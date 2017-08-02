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
	"context"
	// "encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/datatypes"
	// sql "github.com/qri-io/dataset_sql"
	"github.com/spf13/cobra"

	// bstore "github.com/ipfs/go-ipfs/blocks/blockstore"
	blockservice "github.com/ipfs/go-ipfs/blockservice"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreunix"
	dag "github.com/ipfs/go-ipfs/merkledag"
	// dagtest "github.com/ipfs/go-ipfs/merkledag/test"
	files "github.com/ipfs/go-ipfs/commands/files"
	fsrepo "github.com/ipfs/go-ipfs/repo/fsrepo"
)

func runQuery() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo, err := fsrepo.Open("~/ipfs")
	ExitIfErr(err)

	cfg := &core.BuildCfg{
		Repo:   repo,
		Online: false,
	}

	node, err := core.NewNode(ctx, cfg)
	ExitIfErr(err)

	bserv := blockservice.New(node.Blockstore, node.Exchange)
	dagserv := dag.NewDAGService(bserv)

	fileAdder, err := coreunix.NewAdder(ctx, node.Pinning, node.Blockstore, dagserv)
	ExitIfErr(err)

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

	err = ioutil.WriteFile("testdata/resource.json", rdata, os.ModePerm)
	ExitIfErr(err)

	fi, err := os.Stat("testdata/resource.json")
	ExitIfErr(err)

	rfile, err := files.NewSerialFile("resource.json", "testdata/resource.json", false, fi)
	ExitIfErr(err)

	outChan := make(chan interface{}, 8)

	fileAdder.Out = outChan
	go func() {
		defer close(outChan)
		for {
			select {
			case out, ok := <-outChan:
				if ok {
					output := out.(*coreunix.AddedObject)
					if len(output.Hash) > 0 {
						fmt.Printf("added %s", output.Hash)
						return
					}
				}
			}
		}
	}()

	err = fileAdder.AddFile(rfile)
	ExitIfErr(err)

	_, err = fileAdder.Finalize()
	ExitIfErr(err)

	err = fileAdder.PinRoot()
	ExitIfErr(err)
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a query",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		runQuery()
		// repo, err := fsrepo.Open("~/ipfs")
		// ExitIfErr(err)

		// if len(args) == 0 {
		// 	ErrExit(fmt.Errorf("Please provide a query or address to execute"))
		// }

		// q := &dataset.Query{}
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
