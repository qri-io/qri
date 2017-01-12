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

	"github.com/spf13/viper"

	"github.com/qri-io/dataset"
	"github.com/qri-io/namespace"
	"github.com/qri-io/namespace/json_api"
	"github.com/spf13/cobra"
)

// remoteCmd represents the namespace command
var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "List & edit remotes",
	// Long:  `Namespaces are a domain connected with a base address.`,
}

var remoteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List remotes",
	Run: func(cmd *cobra.Command, args []string) {
		for i, r := range GetRemotes(cmd, args) {
			PrintInfo("%d. %s", i+1, r.Namespace().Url())
		}
	},
}

var remoteSetTokenCmd = &cobra.Command{
	Use:   "set-token",
	Short: "Set access token for a remote",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			ErrExit(fmt.Errorf("require [remote-name] [access-token] as arguments"))
		}

		rmt := GetRemotes(cmd, args)
		for _, r := range rmt {
			if r.Url == args[0] {
				r.AccessToken = args[1]
				ExitIfErr(WriteConfigFile(&Config{
					Remotes: rmt,
					Folders: GetFolders(cmd, args),
				}))
			}
		}
	},
}

var remoteAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a remote",
	Run: func(cmd *cobra.Command, args []string) {
		PrintNotYetFinished(cmd)
	},
}

var remoteRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a remote",
	Run: func(cmd *cobra.Command, args []string) {
		PrintNotYetFinished(cmd)
	},
}

func init() {
	remoteCmd.AddCommand(remoteListCmd)
	remoteCmd.AddCommand(remoteAddCmd)
	remoteCmd.AddCommand(remoteRemoveCmd)
	remoteCmd.AddCommand(remoteSetTokenCmd)
	RootCmd.AddCommand(remoteCmd)
}

type Remote struct {
	Url         string          `json:"url"`
	Address     dataset.Address `json:"address"`
	AccessToken string          `json:"access_token"`
}

func (r *Remote) Namespace() namespace.Namespace {
	return json_api.NewNamespace(r.Url, r.Address.String(), r.AccessToken)
}

// Namespaces reads the list of remotes from the config
func GetRemotes(cmd *cobra.Command, args []string) []*Remote {
	namespaceList := viper.Get("remotes")
	if nsSlice, ok := namespaceList.([]interface{}); ok {
		remotes := []*Remote{}
		for _, nsI := range nsSlice {
			if ns, ok := nsI.(map[string]interface{}); ok {
				url := iFaceStr(ns["url"])
				adr := iFaceStr(ns["address"])
				access_token := iFaceStr(ns["access_token"])

				remotes = append(remotes, &Remote{Url: url, Address: dataset.NewAddress(adr), AccessToken: access_token})
			} else {
				ErrExit(fmt.Errorf("invalid remotes configuration. Check your config file!"))
			}
		}
		return remotes
	} else {
		ErrExit(fmt.Errorf("invalid remotes configuration. Check your config file!"))
	}
	return nil
}

func RemoteNamespaces(cmd *cobra.Command, args []string) (ns Namespaces) {
	for _, r := range GetRemotes(cmd, args) {
		ns = append(ns, r.Namespace())
	}
	return
}
