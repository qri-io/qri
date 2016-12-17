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
	"os"

	"github.com/qri-io/dataset"

	query "github.com/qri-io/dataset_sql"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a query",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			ErrExit(fmt.Errorf("Please provide a query or address to execute"))
		}
		stmt, err := query.Parse(args[0])
		ExitIfErr(err)

		adr := dataset.NewAddress("")
		if save := cmd.Flag("save").Value.String(); save != "" {
			if !dataset.ValidAddressString(save) {
				PrintErr(fmt.Errorf("'%s' is not a valid address string to save to", save))
				os.Exit(-1)
			}
			adr = dataset.NewAddress(save)
		}

		format, err := dataset.ParseDataFormatString(cmd.Flag("format").Value.String())
		if err != nil {
			ErrExit(fmt.Errorf("invalid data format: %s", cmd.Flag("format").Value.String()))
		}

		results, data, err := stmt.Exec(GetNamespaces(cmd, args), func(o *query.ExecOpt) {
			o.Format = format
		})
		ExitIfErr(err)

		if !adr.IsEmpty() {
			store := Cache()
			results.Address = adr
			// TODO - need "writeDataset" func
			store.Write(adr.String()+".csv", results.Data)
			PrintSuccess("results saved to: %s", adr.String()+".csv")
			os.Exit(0)
		}

		PrintResults(results, data, format)
	},
}

func init() {
	RootCmd.AddCommand(runCmd)
	runCmd.Flags().StringP("save", "s", "", "save the resulting dataset to a given address")
	runCmd.Flags().StringP("format", "f", "csv", "set output format [csv,json]")
}
