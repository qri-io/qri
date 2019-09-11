package cmd

import (
	"fmt"
	"os"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewFSICommand creates a new `qri fsi` command for working with file system
// integration
func NewFSICommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &FSIOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:    "fsi",
		Hidden: true,
		Short:  "file system integration tools",
	}

	link := &cobra.Command{
		Use:   "link",
		Short: "link a .qri-ref",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Link()
		},
	}

	unlink := &cobra.Command{
		Use:   "unlink",
		Short: "unlink a .qri-ref",
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
	FSIMethods *lib.FSIMethods
}

// Complete adds any missing configuration that can only be added just before
// calling Run
func (o *FSIOptions) Complete(f Factory, args []string) (err error) {
	if len(args) < 1 {
		return fmt.Errorf("please provide the name of a dataset to unlink")
	}
	if o.Refs, err = GetCurrentRefSelect(f, args, -1); err != nil {
		return
	}

	o.FSIMethods, err = f.FSIMethods()
	return err
}

// Link creates a FSI link
func (o *FSIOptions) Link() (err error) {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	p := &lib.LinkParams{
		Dir: pwd,
		Ref: o.Refs.Ref(),
	}
	var res string
	if err := o.FSIMethods.CreateLink(p, &res); err != nil {
		return err
	}

	printSuccess(o.Out, "created dataset reference: %s", res)
	return nil
}

// Unlink executes the fsi unlink command
func (o *FSIOptions) Unlink() error {
	var res string

	for _, ref := range o.Refs.RefList() {
		printRefSelect(o.Out, o.Refs)

		p := &lib.LinkParams{
			Dir: o.Refs.Dir(),
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
