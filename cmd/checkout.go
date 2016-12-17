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
	"github.com/qri-io/history"
	"github.com/spf13/cobra"
)

// checkoutCmd represents the checkout command
var checkoutCmd = &cobra.Command{
	Use:   "checkout",
	Short: "Load history at a designated commit",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		store := Store(cmd, args)
		commitSha1, err := history.StringSha1(args[0])
		if err != nil {
			ErrExit(err)
		}

		if err := history.Checkout(store, store, commitSha1); err != nil {
			ErrExit(err)
		}

		PrintSuccess("checked out: %s", args[0])
	},
}

func init() {
	// RootCmd.AddCommand(checkoutCmd)
}
