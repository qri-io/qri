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
	"github.com/qri-io/qri/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	serverCmdPort string
)

// serverCmd represents the run command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "start a qri server",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		s, err := server.New(func(cfg *server.Config) {
			cfg.NamespaceGraphPath = viper.GetString(NamespaceGraphPath)
			cfg.QueryResultsGraphPath = viper.GetString(QueryResultsGraphPath)
			cfg.ResourceQueriesGraphPath = viper.GetString(ResourceQueriesGraphPath)
			cfg.ResourceMetaGraphPath = viper.GetString(ResourceMetaGraphPath)
			cfg.Port = serverCmdPort
		})
		ExitIfErr(err)

		err = s.Serve()
		ExitIfErr(err)
	},
}

func init() {
	serverCmd.Flags().StringVarP(&serverCmdPort, "port", "p", "3000", "port to start server on")
	RootCmd.AddCommand(serverCmd)
}
