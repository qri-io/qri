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
	// "github.com/qri-io/history"
	lns "github.com/qri-io/namespace/local"
	"github.com/qri-io/namespace/remote"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Update remote repository with local datasets",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		// store := Store(cmd, args)
		// err := history.Push(store, "localhost:5380")
		// ExitIfErr(err)

		adr := GetAddress(cmd, args)
		ns := lns.NewNamespaceFromPath(GetWd())
		zr, size, err := ns.Package(adr)
		if err != nil {
			ErrExit(fmt.Errorf("couldn't build local package for address: %s", adr.String()))
		}

		rmt := remote.New("localhost", "qri")
		err = rmt.WritePackage(adr, zr, size)
		if err != nil {
			ErrExit(err)
		}

		PrintSuccess("sucessfully pushed")
	},
}

func init() {
	RootCmd.AddCommand(pushCmd)
}
