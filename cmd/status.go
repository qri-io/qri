package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewStatusCommand creates a `qri status` command that compares working directory to prev version
func NewStatusCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &StatusOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "status",
		Short:   "Show status of working directory",
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

// StatusOptions encapsulates state for the Status command
type StatusOptions struct {
	ioes.IOStreams

	Refs *RefSelect
	Dir  string

	FSIMethods *lib.FSIMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *StatusOptions) Complete(f Factory, args []string) (err error) {
	o.Refs, err = GetLinkedRefSelect()
	if err != nil {
		return err
	}
	o.FSIMethods, err = f.FSIMethods()
	return
}

// Run executes the status command
func (o *StatusOptions) Run() (err error) {
	printRefSelect(o.Out, o.Refs)

	res := []lib.StatusItem{}
	if err := o.FSIMethods.Status(&o.Dir, &res); err != nil {
		printErr(o.ErrOut, err)
		return nil
	}

	clean := true
	valid := true
	for _, si := range res {
		switch si.Type {
		case fsi.STRemoved:
			printErr(o.Out, fmt.Errorf("  %s:  %s", si.Type, si.Component))
			clean = false
		case fsi.STUnmodified:
			// noop
		default:
			printErr(o.Out, fmt.Errorf("  %s: %s (source: %s)", si.Type, si.Component, filepath.Base(si.SourceFile)))
			clean = false
		}
		// TODO(dlong): Validate each file / component, set `valid` to false if any problems exist
	}

	if clean {
		printSuccess(o.Out, "working directory clean")
	} else if valid {
		printSuccess(o.Out, "\nrun `qri save` to commit this dataset")
	}
	return nil
}
