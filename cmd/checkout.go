package cmd

import (
	"fmt"
	"strings"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
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

	Refs *RefSelect
	Args []string

	FSIMethods *lib.FSIMethods
}

// Complete completes a the command
func (o *CheckoutOptions) Complete(f Factory, args []string) (err error) {
	o.Refs, err = GetCurrentRefSelect(f, []string{}, 1)
	if err == repo.ErrEmptyRef {
		// Not an error, must be a fresh checkout.
		o.Refs = NewExplicitRefSelect("")
	} else if err != nil {
		return err
	}
	o.Args = args
	o.FSIMethods, err = f.FSIMethods()
	return err
}

// Run executes the `checkout` command
func (o *CheckoutOptions) Run() (err error) {
	var freshDatasetRef, refPath, componentName, folderName string

	// Process arguments to get dataset name, component name, and/or ref path.
	for _, arg := range o.Args {
		if strings.HasPrefix(arg, "@/ipfs/") {
			if refPath != "" {
				return fmt.Errorf("cannot provide more than one ref Path")
			}
			refPath = arg
			continue
		}

		if isDatasetField.MatchString(arg) {
			if componentName != "" {
				return fmt.Errorf("cannot provide more than one dataset field")
			}
			componentName = arg
			continue
		}

		pos := strings.Index(arg, "/")
		if pos > -1 {
			if freshDatasetRef != "" {
				return fmt.Errorf("cannot provide more than one dataset name")
			}
			freshDatasetRef = arg
			// Derive directory name from the dataset name.
			folderName = freshDatasetRef[pos+1:]
			continue
		}

		return fmt.Errorf("unknown argument \"%s\"", arg)
	}

	if !o.Refs.IsLinked() {
		// Fresh checkout
		if freshDatasetRef == "" {
			return fmt.Errorf("TODO A")
		}
		if refPath != "" || componentName != "" {
			return fmt.Errorf("fresh checkout of a dataset can't have a path or component name")
		}

		if err = qfs.AbsPath(&folderName); err != nil {
			return err
		}

		var res string
		err = o.FSIMethods.Checkout(&lib.CheckoutParams{
			Dir: folderName,
			Ref: freshDatasetRef,
		}, &res)
		if err != nil {
			return err
		}
		printSuccess(o.Out, "created and linked working directory %s for existing dataset",
			folderName)
	} else {
		// Historic checkout
		if freshDatasetRef != "" {
			return fmt.Errorf("TODO B")
		}
		if refPath == "" && componentName == "" {
			return fmt.Errorf("historic checkout needs either a path or component name")
		}

		ref := o.Refs.Ref()
		if refPath != "" {
			ref += refPath
		}

		var res string
		err = o.FSIMethods.CheckoutHistoric(&lib.CheckoutParams{
			Ref:       ref,
			Component: componentName,
		}, &res)
		if err != nil {
			return err
		}
		printSuccess(o.Out, "TODO dataset")
	}
	return nil
}
