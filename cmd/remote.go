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

	"github.com/spf13/cobra"
)

// remoteCmd represents the remote command
var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Run: func(cmd *cobra.Command, args []string) {
	// 	// TODO: Work your own magic here
	// 	fmt.Println("remote called")
	// },
}

var remoteListCmd = &cobra.Command{
	Use:   "list",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		remotes, err := GetRepo(cmd, args).Remotes()
		if err != nil {
			ErrExit(err)
		}

		fmt.Println(remotes)
	},
}

var remoteAddCmd = &cobra.Command{
	Use:   "add",
	Short: "add a remote repository",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		r, err := GetRepo(cmd, args).AddRemote(args[0], args[1])
		if err != nil {
			ErrExit(err)
		}

		fmt.Printf("added remote: %s", r.Name)
	},
}

var remoteRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if err := GetRepo(cmd, args).RemoveRemote(args[0]); err != nil {
			ErrExit(err)
		}

		fmt.Println("removed remote: %s", args[0])
	},
}

var remoteSetUrlCmd = &cobra.Command{
	Use:   "set-url",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		r := GetRepo(cmd, args)

		remote, err := r.RemoteForName(args[0])
		if err != nil {
			ErrExit(err)
		}

		remote.Url = args[1]
		if err = r.UpdateRemote(remote); err != nil {
			ErrExit(err)
		}

		fmt.Println("%s:%s", remote.Name, remote.Url)
	},
}

func init() {
	remoteCmd.AddCommand(remoteListCmd)
	remoteCmd.AddCommand(remoteAddCmd)
	remoteCmd.AddCommand(remoteRemoveCmd)
	remoteCmd.AddCommand(remoteSetUrlCmd)

	RootCmd.AddCommand(remoteCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// remoteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// remoteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
