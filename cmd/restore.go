package cmd

import (
	"fmt"
	"strings"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewRestoreCommand creates new `qri restore` command
func NewRestoreCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &RestoreOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "restore",
		Short:   "restore returns part or all of a dataset to a previous state",
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

// RestoreOptions encapsulates state for the `restore` command
type RestoreOptions struct {
	ioes.IOStreams

	Refs          *RefSelect
	Path          string
	ComponentName string

	FSIMethods *lib.FSIMethods
}

// Complete configures the restore command
func (o *RestoreOptions) Complete(f Factory, args []string) (err error) {
	dsRefList := []string{}
	o.Path = ""
	o.ComponentName = ""

	// TODO(dlong): Add low-level utilities that parse strings like "peername/ds_name", and
	// "/ipfs/QmFoo", "meta.description", etc and use those everywhere. Use real regexs so
	// that we properly handle user input everywhere. Too much code is duplicating half working
	// input handling for various stringified identifiers.

	// Process arguments to get dataset name, component name, and/or ref path.
	for _, arg := range args {
		if strings.HasPrefix(arg, "/ipfs/") {
			if o.Path != "" {
				return fmt.Errorf("cannot provide more than one ref Path")
			}
			o.Path = arg
			continue
		}

		// Treat "schema" as "structure.schema"
		if arg == "schema" {
			arg = "structure.schema"
		}

		if isDatasetField.MatchString(arg) {
			if o.ComponentName != "" {
				return fmt.Errorf("cannot provide more than one dataset field")
			}
			o.ComponentName = arg
			continue
		}

		pos := strings.Index(arg, "/")
		if pos > -1 {
			if len(dsRefList) != 0 {
				return fmt.Errorf("cannot provide more than one dataset name")
			}
			dsRefList = []string{arg}
			continue
		}

		return fmt.Errorf("unknown argument \"%s\"", arg)
	}

	o.Refs, err = GetCurrentRefSelect(f, dsRefList, 1)
	if err != nil {
		return err
	}
	o.FSIMethods, err = f.FSIMethods()
	return err
}

// Run executes the `restore` command
func (o *RestoreOptions) Run() (err error) {
	printRefSelect(o.Out, o.Refs)

	ref := o.Refs.Ref()
	if o.Path != "" {
		ref += o.Path
	}

	var res string
	err = o.FSIMethods.Restore(&lib.RestoreParams{Ref: ref, Component: o.ComponentName}, &res)
	if err != nil {
		return err
	}
	if o.ComponentName != "" && o.Path == "" {
		printSuccess(o.Out, fmt.Sprintf("Restored %s of dataset %s", o.ComponentName, ref))
	} else if o.Path != "" && o.ComponentName == "" {
		printSuccess(o.Out, fmt.Sprintf("Restored dataset version %s", ref))
	}
	// TODO(dlong): Print message when both component and path are specified.
	return nil
}
