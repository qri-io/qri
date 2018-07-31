package cmd

import (
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

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
		RunE: func(cmd *cobra.Command, args []string) error {
			printInfo(ioStreams.Out, lib.VersionNumber)
			return nil
		},
	}
	return cmd
}
