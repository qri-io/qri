package cmd

import (
	"context"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewRenameCommand creates a new `qri rename` cobra command for renaming datasets
func NewRenameCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &RenameOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "rename",
		Aliases: []string{"mv"},
		Short:   "change the name of a dataset",
		Long: `Rename changes the name of a dataset.

Note that if someone has added your dataset to their qri node, and then
you rename your local dataset, your peer's version of your dataset will
not have the updated name. While this won't break anything, it will
confuse anyone who has added your dataset before the change. Try to keep
renames to a minimum.`,
		Example: `  # Rename a dataset named annual_pop to annual_population:
  $ qri rename me/annual_pop me/annual_population`,
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

// RenameOptions encapsulates state for the rename command
type RenameOptions struct {
	ioes.IOStreams

	From string
	To   string

	DatasetMethods *lib.DatasetMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *RenameOptions) Complete(f Factory, args []string) (err error) {
	if len(args) == 2 {
		o.From = args[0]
		o.To = args[1]
	}
	o.DatasetMethods, err = f.DatasetMethods()
	return
}

// Validate checks that all user input is valid
func (o *RenameOptions) Validate() error {
	if o.From == "" || o.To == "" {
		return errors.New(lib.ErrBadArgs, "please provide two dataset names, the original and the new name, for example:\n    $ qri rename me/old_name me/new_name\nsee `qri rename --help` for more details")
	}
	return nil
}

// Run executes the rename command
func (o *RenameOptions) Run() error {
	p := &lib.RenameParams{
		Current: o.From,
		Next:    o.To,
	}
	ctx := context.TODO()
	res, err := o.DatasetMethods.Rename(ctx, p)
	if err != nil {
		return err
	}

	printSuccess(o.Out, "renamed dataset to %s", res.Name)
	return nil
}
