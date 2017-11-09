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
