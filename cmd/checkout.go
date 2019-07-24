package cmd

import (
	"fmt"
	"strings"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewCheckoutCommand creates new `qri checkout` command that connects a working directory in
// the local filesystem to a dataset your repo.
func NewCheckoutCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &CheckoutOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "checkout",
		Short:   "checkout created a linked directory and writes dataset files to that directory",
		Long:    ``,
		Example: ``,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Complete(f, args)
			return o.Run()
		},
	}

	return cmd
}

// CheckoutOptions encapsulates state for the `checkout` command
type CheckoutOptions struct {
	ioes.IOStreams

	Args []string

	FSIMethods *lib.FSIMethods
}

// Complete completes a the command
func (o *CheckoutOptions) Complete(f Factory, args []string) (err error) {
	o.Args = args
	o.FSIMethods, err = f.FSIMethods()
	return err
}

// Run executes the `checkout` command
func (o *CheckoutOptions) Run() (err error) {
	// TODO: Finalize UI for command-line checkout command.
	ref := o.Args[0]

	// Derive directory name from the dataset name.
	pos := strings.Index(ref, "/")
	if pos == -1 {
		return fmt.Errorf("expect '/' in dataset ref")
	}
	folderName := ref[pos+1:]

	if err = qfs.AbsPath(&folderName); err != nil {
		return err
	}

	var res string
	err = o.FSIMethods.Checkout(&lib.CheckoutParams{Dir: folderName, Ref: ref}, &res)
	if err != nil {
		return err
	}

	printSuccess(o.Out, "created and linked working directory %s for existing dataset", folderName)
	return nil
}
