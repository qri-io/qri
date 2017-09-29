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
	serverMemOnly bool
	serverOffline bool
)

// serverCmd represents the run command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "start a qri server",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		PrintInfo(viper.GetString(QriRepoPath))
		s, err := server.New(func(cfg *server.Config) {
			cfg.Port = serverCmdPort
			cfg.MemOnly = serverMemOnly
			cfg.QriRepoPath = viper.GetString(QriRepoPath)
			cfg.FsStorePath = viper.GetString(IpfsFsPath)
			cfg.Online = !serverOffline
		})
		ExitIfErr(err)

		err = s.Serve()
		ExitIfErr(err)
	},
}

func init() {
	serverCmd.Flags().StringVarP(&serverCmdPort, "port", "p", "3000", "port to start server on")
	serverCmd.Flags().BoolVarP(&serverMemOnly, "mem-only", "", false, "run qri entirely in-memory")
	serverCmd.Flags().BoolVarP(&serverOffline, "offline", "", false, "disable networking")
	RootCmd.AddCommand(serverCmd)
}
