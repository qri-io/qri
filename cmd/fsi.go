package cmd

import (
	"path/filepath"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewFSICommand creates a new `qri fsi` command for working with file system
// integration
func NewFSICommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &FSIOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "workdir",
		Aliases: []string{"fsi"},
		Short:   "file system integration tools",
		Annotations: map[string]string{
			"group": "workdir",
		},
	}

	link := &cobra.Command{
		Use:   "link DATASET PATH",
		Short: "link a dataset to a directory on disk",
		Example: `  # Link a dataset to the current working directory:
  $ qri workdir link peername/dataset .`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Link()
		},
	}

	unlink := &cobra.Command{
		Use:   "unlink DATASET",
		Short: "unlink a dataset from a directory on disk",
		// Use max instead of exact args so we can provide a nicer error.
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Unlink()
		},
	}

	cmd.AddCommand(link, unlink)
	return cmd
}

// FSIOptions encapsulates state for the dag command
type FSIOptions struct {
	ioes.IOStreams

	Refs       *RefSelect
	Path       string
	FSIMethods *lib.FSIMethods
}

// Complete adds any missing configuration that can only be added just before
// calling Run
func (o *FSIOptions) Complete(f Factory, args []string) (err error) {
	o.FSIMethods, err = f.FSIMethods()
	if err != nil {
		return err
	}
	if o.Refs, err = GetCurrentRefSelect(f, args, 1, o.FSIMethods); err != nil {
		return err
	}

	if len(args) > 1 {
		o.Path = args[1]
	}

	return nil
}

// Link creates a FSI link
func (o *FSIOptions) Link() (err error) {
	o.Path, err = filepath.Abs(o.Path)
	if err != nil {
		return err
	}

	p := &lib.LinkParams{
		Dir: o.Path,
		Ref: o.Refs.Ref(),
	}
	res := &dsref.VersionInfo{}
	if err := o.FSIMethods.CreateLink(p, res); err != nil {
		return err
	}

	printSuccess(o.Out, "created dataset reference: %s", res.SimpleRef().Human())
	return nil
}

// Unlink executes the fsi unlink command
func (o *FSIOptions) Unlink() error {
	var res string

	for _, ref := range o.Refs.RefList() {
		printRefSelect(o.ErrOut, o.Refs)

		p := &lib.LinkParams{
			Ref: ref,
		}
		if err := o.FSIMethods.Unlink(p, &res); err != nil {
			printErr(o.ErrOut, err)
			return nil
		}

		printSuccess(o.Out, "unlinked: %s", res)
	}
	return nil
}
