package cmd

import (
	"github.com/spf13/cobra"
)

var datasetListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "list your local datasets",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		refs, err := GetRepo(false).Namespace(100, 0)
		ExitIfErr(err)
		for _, ref := range refs {
			PrintInfo("%s\t\t\t: %s", ref.Name, ref.Path)
		}
	},
}

func init() {
	RootCmd.AddCommand(datasetListCmd)
}
