package cmd

import (
	"fmt"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewRemoveCommand creates a new `qri remove` cobra command for removing datasets from a local repository
func NewRemoveCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &RemoveOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "remove [DATASET]",
		Aliases: []string{"rm", "delete"},
		Short:   "remove a dataset from your local repository",
		Long: `Remove gets rid of a dataset from your qri node. After running remove, qri will
no longer list your dataset as being available locally. By default, remove frees
up the space taken up by the dataset, but not right away. The IPFS repo that’s
storing the data will need to garbage-collect that data when it’s good & ready,
which could be anytime. If you’re running low on space, garbage collection will
be sooner.

Keep in mind that by default your IPFS repo is capped at 10GB in size, if you
adjust this cap using IPFS, qri will respect it.

In the future we’ll add a flag that’ll force immediate removal of a dataset from
both qri & IPFS. Promise.`,
		Example: `  # Remove a dataset named ` + "`annual_pop`" + `:
  $ qri remove me/annual_pop --all`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		// Use *max* so we can print a nicer message for no or malformed args
		Args: cobra.MaximumNArgs(1),
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

	cmd.Flags().StringVarP(&o.RevisionsText, "revisions", "r", "", "revisions to delete")
	cmd.Flags().BoolVarP(&o.All, "all", "a", false, "synonym for --revisions=all")
	cmd.Flags().BoolVar(&o.KeepFiles, "keep-files", false, "don't modify files in working directory")
	cmd.Flags().BoolVarP(&o.Force, "force", "f", false, "remove files even if dirty")

	return cmd
}

// RemoveOptions encapsulates state for the remove command
type RemoveOptions struct {
	ioes.IOStreams

	Refs *RefSelect

	RevisionsText string
	Revision      dsref.Rev
	All           bool
	KeepFiles     bool
	Force         bool

	DatasetMethods *lib.DatasetMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *RemoveOptions) Complete(f Factory, args []string) (err error) {
	if o.DatasetMethods, err = f.DatasetMethods(); err != nil {
		return err
	}
	if o.Refs, err = GetCurrentRefSelect(f, args, -1, nil); err != nil {
		// This error will be handled during validation
		if err != repo.ErrEmptyRef {
			return err
		}
		err = nil
	}
	if o.All {
		o.Revision = dsref.NewAllRevisions()
	} else {
		if o.RevisionsText == "" {
			return fmt.Errorf("need --all or --revisions to specify how many revisions to remove")
		}
		revisions, err := dsref.ParseRevs(o.RevisionsText)
		if err != nil {
			return err
		}
		if len(revisions) != 1 {
			return fmt.Errorf("need exactly 1 revision parameter to remove")
		}
		if revisions[0] == nil {
			return fmt.Errorf("invalid nil revision")
		}
		o.Revision = *revisions[0]
	}
	return err
}

// Validate checks that all user input is valid
func (o *RemoveOptions) Validate() error {
	if o.Refs.Ref() == "" {
		return errors.New(lib.ErrBadArgs, "please specify a dataset path or name you would like to remove from your qri node")
	}
	return nil
}

// Run executes the remove command
func (o *RemoveOptions) Run() (err error) {
	printRefSelect(o.ErrOut, o.Refs)

	params := lib.RemoveParams{
		Ref:       o.Refs.Ref(),
		Revision:  o.Revision,
		KeepFiles: o.KeepFiles,
		Force:     o.Force,
	}

	res := lib.RemoveResponse{}
	if err = o.DatasetMethods.Remove(&params, &res); err != nil {
		if err == repo.ErrNotFound {
			return errors.New(err, fmt.Sprintf("could not find dataset '%s'", o.Refs.Ref()))
		}
		if err == lib.ErrCantRemoveDirectoryDirty {
			printErr(o.ErrOut, err)
			printErr(o.ErrOut, fmt.Errorf("use either --keep-files, or --force"))
			return fmt.Errorf("dataset not removed")
		}
		return err
	}

	if res.NumDeleted == dsref.AllGenerations {
		printSuccess(o.Out, "removed entire dataset '%s'", res.Ref)
	} else if res.NumDeleted != 0 {
		printSuccess(o.Out, "removed %d revisions of dataset '%s'", res.NumDeleted, res.Ref)
	} else if res.Message != "" {
		printSuccess(o.Out, "removed remains of dataset from %s", res.Message)
	}
	if res.Unlinked {
		printSuccess(o.Out, "removed dataset link")
	}
	return nil
}
