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

	"github.com/qri-io/namespace"

	"github.com/spf13/cobra"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Update remote repository with local datasets",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		adr := GetAddress(cmd, args)

		lns := LocalNamespaces(cmd, args)
		zr, size, err := lns.Package(adr)
		if err != nil {
			ErrExit(fmt.Errorf("couldn't build local package for address: %s", adr.String()))
		}

		for _, rmt := range GetRemotes(cmd, args) {
			if rmt.Address.IsAncestor(adr) {
				rns := rmt.Namespace()
				if writableNs, ok := rns.(namespace.WritableNamespace); ok {
					spinner.Start()
					err = writableNs.WritePackage(adr, zr, size)
					spinner.Stop()
					if err != nil {
						ErrExit(err)
					}
					PrintSuccess("sucessfully pushed")
					return
				} else {
					PrintErr(fmt.Errorf("remote: %s isn't writable.", rns.Url()))
					return
				}
			}
		}

		PrintErr(fmt.Errorf("couldn't find a place to push address: %s. are your remotes configured properly?", adr.String()))
	},
}

func init() {
	RootCmd.AddCommand(pushCmd)
}
