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
	"bytes"
	"encoding/csv"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	query "github.com/qri-io/dataset_sql"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run a query",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		stmt, err := query.Parse(args[0])
		if err != nil {
			fmt.Printf("statement parse error: %s", err.Error())
			os.Exit(1)
		}

		results, err := stmt.Exec(GetDomainList())
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		fmt.Println()
		table := tablewriter.NewWriter(os.Stdout)
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
		table.SetHeader(results.FieldNames())

		r := csv.NewReader(bytes.NewBuffer(results.Data))
		for {
			rec, err := r.Read()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				fmt.Println(err.Error())
				os.Exit(1)
			}

			table.Append(rec)
		}

		table.Render()
	},
}

func init() {
	RootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
