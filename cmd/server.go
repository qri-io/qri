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
	// ipfs "github.com/qri-io/castore/ipfs"
	// "github.com/qri-io/dataset"
	"github.com/qri-io/qri/server"
	"github.com/spf13/cobra"
)

// serverCmd represents the run command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "start a qri server",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		// var (
		// 	resource *dataset.Resource
		// 	results  []byte
		// )

		// ds, err := ipfs.NewDatastore(func(c *ipfs.StoreCfg) {
		// 	c.Online = true
		// })
		// ExitIfErr(err)

		err := server.New().Serve()
		ExitIfErr(err)
	},
}

func init() {
	RootCmd.AddCommand(serverCmd)
}
