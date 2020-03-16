package cmd

import (
	"fmt"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewShowCommitCommand creates a new `qri search` command that searches for datasets
func NewShowCommitCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &ShowCommitOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "showcommit",
		Short: "Show information about a commit",
		Long: `
The command showcommit will show information about a commit, specifically what
components were added or removed, at a specific version. This is analagous to
the status command, except only available for dataset versions in history.

  # show information about the head commit
  qri showcommit me/dataset_name`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return o.Run()
		},
	}

	return cmd
}

// ShowCommitOptions encapsulates state for the search command
type ShowCommitOptions struct {
	ioes.IOStreams

	Refs       *RefSelect
	FSIMethods *lib.FSIMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ShowCommitOptions) Complete(f Factory, args []string) (err error) {
	if o.FSIMethods, err = f.FSIMethods(); err != nil {
		return err
	}
	o.Refs, err = GetCurrentRefSelect(f, args, 1, o.FSIMethods)
	return nil
}

// Validate checks that any user input is valid
func (o *ShowCommitOptions) Validate() error {
	return nil
}

// Run executes the showcommit command
func (o *ShowCommitOptions) Run() (err error) {
	printRefSelect(o.ErrOut, o.Refs)

	res := []lib.StatusItem{}
	ref := o.Refs.Ref()
	if err := o.FSIMethods.ShowCommit(&ref, &res); err != nil {
		printErr(o.ErrOut, err)
		return nil
	}

	for _, si := range res {
		printInfo(o.Out, fmt.Sprintf("  %s: %s", si.Component, si.Type))
	}

	return nil
}
