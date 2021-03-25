package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewCheckoutCommand creates new `qri checkout` command that connects a working directory in
// the local filesystem to a dataset your repo.
func NewCheckoutCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &CheckoutOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "checkout DATASET",
		Short: "create a linked directory and write dataset files to that directory",
		Long:  ``,
		Example: `  # Place a copy of me/annual_pop in the ./annual_pop directory:
  $ qri checkout me/annual_pop`,
		Annotations: map[string]string{
			"group": "workdir",
		},
		Args: cobra.RangeArgs(1, 2),
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

	Instance *lib.Instance

	Refs *RefSelect
	Dir  string
}

// Complete configures the checkout command
func (o *CheckoutOptions) Complete(f Factory, args []string) (err error) {
	if o.Instance, err = f.Instance(); err != nil {
		return err
	}

	o.Refs, err = GetCurrentRefSelect(f, args, 1, EnsureFSIAgrees(o.Instance))
	if err != nil {
		return err
	}

	if len(args) == 2 {
		o.Dir = args[1]
	} else {
		o.Dir = ""
	}
	return nil
}

// Run executes the `checkout` command
func (o *CheckoutOptions) Run() (err error) {
	ctx := context.TODO()
	inst := o.Instance

	if !o.Refs.IsExplicit() {
		return fmt.Errorf("checkout requires an explicitly provided dataset ref")
	}
	ref := o.Refs.Ref()

	// Derive directory name from the dataset name.
	pos := strings.Index(ref, "/")
	if pos == -1 {
		return fmt.Errorf("expect '/' in dataset ref")
	}
	if o.Dir == "" {
		// Dataset names should always be safe to use for directories, since they use a small
		// subset of characters. However, it's possible the user has bad data in their repo, so
		// generate a name just to be safe.
		o.Dir = dsref.GenerateName(ref[pos+1:], "")
	}

	if err = inst.Filesys().Checkout(ctx, &lib.LinkParams{Dir: o.Dir, Refstr: ref}); err != nil {
		return err
	}
	printSuccess(o.Out, "created and linked working directory %s for existing dataset", o.Dir)
	return nil
}
