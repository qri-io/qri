package cmd

import (
	"fmt"
	"strings"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/varName"
	"github.com/spf13/cobra"
)

// NewCheckoutCommand creates new `qri checkout` command that connects a working directory in
// the local filesystem to a dataset your repo.
func NewCheckoutCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &CheckoutOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "checkout",
		Short:   "checkout creates a linked directory and writes dataset files to that directory",
		Long:    ``,
		Example: ``,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	return cmd
}

// CheckoutOptions encapsulates state for the `checkout` command
type CheckoutOptions struct {
	ioes.IOStreams

	Refs *RefSelect

	FSIMethods *lib.FSIMethods
}

// Complete configures the checkout command
func (o *CheckoutOptions) Complete(f Factory, args []string) (err error) {
	o.Refs, err = GetCurrentRefSelect(f, args, 1)
	if err != nil {
		return err
	}
	o.FSIMethods, err = f.FSIMethods()
	return err
}

// Run executes the `checkout` command
func (o *CheckoutOptions) Run() (err error) {
	if !o.Refs.IsExplicit() {
		return fmt.Errorf("checkout requires an explicitly provided dataset ref")
	}
	ref := o.Refs.Ref()

	// Derive directory name from the dataset name.
	pos := strings.Index(ref, "/")
	if pos == -1 {
		return fmt.Errorf("expect '/' in dataset ref")
	}
	folderName := varName.CreateVarNameFromString(ref[pos+1:])

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
