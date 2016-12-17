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
	"github.com/qri-io/dataset"
	"github.com/qri-io/namespace/local"
	"github.com/spf13/cobra"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "List errors in a dataset",
	Long: `Validate will load the dataset in question
and check each of it's rows against the constraints listed
in the dataset's fields.

For a full rundown on validation visit:
http://docs.qri.io/concepts/validation`,
	Run: func(cmd *cobra.Command, args []string) {
		// store := Store(cmd, args)
		// errs, err := history.Validate(store)
		// ExitIfErr(err)

		adr := GetAddress(cmd, args)
		ns := local.NewNamespaceFromPath(GetWd())
		ds, err := ns.Dataset(adr)
		ExitIfErr(err)

		if cmd.Flag("check-links").Value.String() == "true" {
			validation, data, count, err := ds.ValidateDeadLinks(Cache())
			ExitIfErr(err)
			if count > 0 {
				PrintResults(validation, data, dataset.CsvDataFormat)
			} else {
				PrintSuccess("✔ All good!")
			}
		}

		validation, data, count, err := ds.ValidateData(Cache())
		ExitIfErr(err)
		if count > 0 {
			PrintResults(validation, data, dataset.CsvDataFormat)
		} else {
			PrintSuccess("✔ All good!")
		}
	},
}

func init() {
	RootCmd.AddCommand(validateCmd)
	validateCmd.Flags().BoolP("check-links", "l", false, "check dead links")
}
