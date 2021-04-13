package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	qerr "github.com/qri-io/qri/errors"
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
		Long: `Remove deletes datasets from qri.

For read-only datasets you've pulled from others, Remove gets rid of a dataset 
from your qri node. After running remove, qri will no longer list that dataset 
as being available locally, and may free up the storage space.

For datasets you can edit, remove deletes commits from a dataset history.
Use delete to "correct the record" by erasing commits. Running remove on 
writable datasets requires a '--revisions' flag, specifying the number of 
commits to delete. Remove always starts from the latest (HEAD) commit, working 
backwards toward the first commit.

Remove can also be used to ask remotes to delete datasets with the '--remote'
flag. Passing the remote flag will run the operation as a network request,
reporting the results of attempting to remove on the destination remote.
The remote flag can only be used to completely remove a dataset from a remote.
To edit history on a remote, run delete locally and use 'qri push' to send the
updated history to the remote. Any command run with the remote flag has no
effect on local data.`,
		Example: `  # delete a dataset cloned from another user
  $ qri remove user/world_bank_population

  # delete the latest commit from annual_pop
  $ qri remove me/annual_pop --revisions 1

  # delete the latest two versions from history
  $ qri remove me/annual_pop --revisions 2

  # destroy a dataset named 'annual_pop'
  $ qri remove --all me/annual_pop

  # ask the registry to delete a dataset
  $ qri remove --remote registry me/annual_pop`,
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
	cmd.Flags().BoolVarP(&o.Force, "force", "f", false, "remove files even if a working directory is dirty")
	cmd.Flags().StringVar(&o.Remote, "remote", "", "remote address to remove from")

	return cmd
}

// RemoveOptions encapsulates state for the remove command
type RemoveOptions struct {
	ioes.IOStreams

	Refs *RefSelect

	Remote        string
	RevisionsText string
	Revision      *dsref.Rev
	All           bool
	KeepFiles     bool
	Force         bool

	inst *lib.Instance
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *RemoveOptions) Complete(f Factory, args []string) (err error) {
	if o.inst, err = f.Instance(); err != nil {
		return
	}

	if o.Refs, err = GetCurrentRefSelect(f, args, 1, nil); err != nil {
		// This error will be handled during validation
		if err != repo.ErrEmptyRef {
			return
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
		o.Revision = revisions[0]
	}
	return
}

// Validate checks that all user input is valid
func (o *RemoveOptions) Validate() error {
	if o.Refs.Ref() == "" {
		return qerr.New(lib.ErrBadArgs, "please specify a dataset path or name you would like to remove from your qri node")
	}
	return nil
}

// Run executes the remove command
func (o *RemoveOptions) Run() (err error) {
	printRefSelect(o.ErrOut, o.Refs)

	if o.Remote != "" {
		return o.RemoveRemote()
	}

	params := lib.RemoveParams{
		Ref:       o.Refs.Ref(),
		Revision:  o.Revision,
		KeepFiles: o.KeepFiles,
		Force:     o.Force,
	}

	ctx := context.TODO()
	res, err := o.inst.WithSource("local").Dataset().Remove(ctx, &params)
	if err != nil {
		// TODO(b5): move this error handling down into lib
		if errors.Is(err, dsref.ErrRefNotFound) {
			return qerr.New(err, fmt.Sprintf("could not find dataset '%s'", o.Refs.Ref()))
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

// RemoveRemote runs the remove command as a network request to a remote
func (o *RemoveOptions) RemoveRemote() error {
	ctx := context.TODO()
	res, err := o.inst.Remote().Remove(ctx, &lib.PushParams{
		Ref:    o.Refs.Ref(),
		Remote: o.Remote,
	})
	if err != nil {
		return fmt.Errorf("dataset not removed")
	}

	// remove profileID info for cleaner output
	res.ProfileID = ""
	printSuccess(o.Out, "removed dataset %s from remote %s", res, o.Remote)
	return nil
}
