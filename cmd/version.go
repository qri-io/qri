package cmd

import (
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/version"
	"github.com/spf13/cobra"
)

// NewVersionCommand creates a new `qri version` cobra command that prints the current qri version
func NewVersionCommand(_ Factory, ioStreams ioes.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "print the version number",
		Long: `Qri uses semantic versioning.

For updates & further information check https://github.com/qri-io/qri/releases`,
		Annotations: map[string]string{
			"group": "other",
		},
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			printInfo(ioStreams.Out, version.Summary())
			return nil
		},
	}
	return cmd
}
