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
	// "github.com/qri-io/qri/p2p"
	"github.com/spf13/cobra"
)

// peerCommand represents the dataset commands
var peerCommand = &cobra.Command{
	Use:   "peer",
	Short: "peer tools",
	Long:  ``,
}

var peerMsgCommand = &cobra.Command{
	Use:   "message",
	Short: "message a peer node",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ErrExit(fmt.Errorf("message not finished"))
		// addr := args[0]
		// node, err := p2p.NewQriNode()
		// ExitIfErr(err)

		// fmt.Println(node.EncapsulatedAddresses())

		// res, err := node.SendMessage(addr, &p2p.Message{Type: p2p.MtUnknown, Payload: "PING"})
		// ExitIfErr(err)

		// fmt.Println(res)
	},
}

func init() {
	peerCommand.AddCommand(peerMsgCommand)
	RootCmd.AddCommand(peerCommand)
}
