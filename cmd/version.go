package cmd

import (
	"github.com/spf13/cobra"
)

// VersionNumber is the current version qri
const VersionNumber = "0.4.1-dev"

// NewVersionCommand creates a new `qri version` cobra command that prints the current qri version
func NewVersionCommand(_ Factory, ioStreams IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "print the version number",
		Long: `qri uses semantic versioning.
  For updates & further information check https://github.com/qri-io/qri/releases`,
		Annotations: map[string]string{
			"group": "other",
		},
		Run: func(cmd *cobra.Command, args []string) {
			printInfo(ioStreams.Out, VersionNumber)
		},
	}
	return cmd
}
