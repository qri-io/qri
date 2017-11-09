package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"describe"},
	Short:   "Show info about a dataset",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(args)
		if len(args) == 0 {
			fmt.Println("please specify an address to get the info of")
			return
		}

		// ds, err := GetNamespaces(cmd, args).Dataset(dataset.NewAddress(args[0]))
		// ExitIfErr(err)
		// PrintDatasetDetailedInfo(ds)
	},
}

func init() {
	// RootCmd.AddCommand(infoCmd)
}
