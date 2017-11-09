package cmd

import "github.com/spf13/cobra"

const VERSION_NUMBER = "0.1.0-alpha"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long: `qri uses semantic versioning.
	For updates & further information check https://github.com/qri-io/qri/releases`,
	Run: func(cmd *cobra.Command, args []string) {
		PrintInfo(VERSION_NUMBER)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
