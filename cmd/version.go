package cmd

import "github.com/spf13/cobra"

// VersionNumber is the current version qri
const VersionNumber = "0.3.3-dev"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print the version number",
	Long: `qri uses semantic versioning.
	For updates & further information check https://github.com/qri-io/qri/releases`,
	Annotations: map[string]string{
		"group": "other",
	},
	Run: func(cmd *cobra.Command, args []string) {
		printInfo(VersionNumber)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
